package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func StartTunnel(state *config.State, interval time.Duration) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client, err := GetClient(ctx, state.Cfg, state.HomeDir)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()

	socketTunnel, err := NewSocketTunnel(client, state.GetDockerSocketPath(), "/var/run/docker.sock")
	if err != nil {
		return err
	}

	go func() {
		if err := socketTunnel.Start(ctx); err != nil && err != context.Canceled {
			fmt.Printf("[-] Docker socket tunnel failed: %v\n", err)
		}
	}()

	fmt.Println("[+] Connected to VM Engine. Actively monitoring for new container ports...")

	// Keep track of active forwarded ports on the Android host
	activeListeners := make(map[string]net.Listener)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nTearing down active proxies...")
			for _, l := range activeListeners {
				_ = l.Close()
			}
			return nil
		case <-ticker.C:
			// Query /proc/net/tcp inside the VM to get listening ports in hex
			ports, err := getActiveVMPorts(client)
			if err != nil {
				continue
			}

			// Spin up dynamic forwarders for newly discovered container ports
			for port := range ports {
				// Avoid clashing with systemic default ports
				if port == "22" || port == "2375" {
					continue
				}

				if _, exists := activeListeners[port]; !exists {
					localAddr := fmt.Sprintf("localhost:%s", port)
					listener, err := net.Listen("tcp", localAddr)
					if err != nil {
						fmt.Printf("[-] Failed to listen on %s: %v\n", localAddr, err)
						continue
					}
					activeListeners[port] = listener
					fmt.Printf("[+] Auto-Forwarding detected port: %s -> VM Port %s\n", localAddr, port)

					go func(l net.Listener, p string) {
						defer func() { _ = l.Close() }()
						for {
							localConn, err := l.Accept()
							if err != nil {
								return
							}
							go bridgeStream(client, localConn, p)
						}
					}(listener, port)
				}
			}

			// Clean up closed ports
			for port := range activeListeners {
				if !ports[port] {
					_ = activeListeners[port].Close()
					delete(activeListeners, port)
				}
			}
		}
	}
}

// Low overhead parser to fetch active TCP ports inside Alpine
func getActiveVMPorts(client *ssh.Client) (map[string]bool, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	var buf bytes.Buffer
	session.Stdout = &buf

	// Fetch listening sockets filtering out column headers
	err = session.Run("netstat -tnl | awk '/:/{print $4}'")
	if err != nil {
		return nil, err
	}

	openPorts := map[string]bool{}
	lines := strings.SplitSeq(buf.String(), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Find the last colon to correctly identify the port, supporting IPv6 (e.g. :::80)
		idx := strings.LastIndex(line, ":")
		if idx != -1 {
			portStr := line[idx+1:]
			decPort, _ := strconv.Atoi(portStr)
			if decPort > 0 {
				openPorts[portStr] = true
			}
		}
	}
	return openPorts, nil
}

func bridgeStream(client *ssh.Client, localConn net.Conn, port string) {
	defer func() { _ = localConn.Close() }()
	// Use localhost to allow the VM to resolve to 127.0.0.1 or [::1] as appropriate
	remoteConn, err := client.Dial("tcp", "localhost:"+port)
	if err != nil {
		fmt.Printf("[-] Failed to dial remote port %s: %v\n", port, err)
		return
	}
	defer func() { _ = remoteConn.Close() }()

	chDone := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(remoteConn, localConn); chDone <- struct{}{} }()
	go func() { _, _ = io.Copy(localConn, remoteConn); chDone <- struct{}{} }()
	<-chDone
}
