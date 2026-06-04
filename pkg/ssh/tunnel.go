package ssh

import (
	"bufio"
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

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
)

// StartConnForwarder briges the Docker socket from the VM to the host and sets up automatic port-forwarding
func StartConnForwarder(state *config.State, interval time.Duration) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client, err := GetClient(ctx, state)
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

	// Clean up state file on exit
	defer func() {
		_ = os.Remove(state.GetPortMapFile())
	}()

	// Initial update to show SYSTEM ports
	updatePortState(state, activeListeners)

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
			// Query listening ports inside the VM
			ports, err := getActiveVMPorts(client)
			if err != nil {
				// If the error is not temporary, it might mean the connection is dead
				fmt.Printf("[-] Failed to query VM ports: %v\n", err)
				// Small delay to avoid spamming if it's a persistent failure
				time.Sleep(time.Second)
				continue
			}

			changed := false

			// Spin up dynamic forwarders for newly discovered container ports
			for id := range ports {
				// Avoid clashing with systemic default ports
				if id == "22 tcp" || id == "68 udp" || id == "546 udp" {
					continue
				}

				parts := strings.Fields(id)
				if len(parts) == 1 {
					parts = append(parts, "tcp")
				}

				if parts[1] == "udp" {
					// TDOO: implement UDP tunnel
					continue
				}

				if _, exists := activeListeners[id]; !exists {
					localAddr := fmt.Sprintf("localhost:%s", parts[0])
					listener, err := net.Listen(parts[1], localAddr)
					if err != nil {
						fmt.Printf("[-] Failed to listen on %s %s: %v\n", parts[1], localAddr, err)
						continue
					}
					activeListeners[id] = listener
					fmt.Printf("[+] Auto-Forwarding detected %s port: %s -> VM Port %s\n", parts[1], localAddr, parts[0])
					changed = true

					// Create dialer for this port
					dial := func() (io.ReadWriteCloser, error) {
						return client.Dial(parts[1], "localhost:"+parts[0])
					}

					go ServeListener(ctx, listener, dial)
				}
			}

			// Clean up closed ports
			for id := range activeListeners {
				if !ports[id] {
					fmt.Printf("[-] Stopped forwarding port: %s\n", id)
					_ = activeListeners[id].Close()
					delete(activeListeners, id)
					changed = true
				}
			}

			if changed {
				updatePortState(state, activeListeners)
			}
		}
	}
}

func updatePortState(s *config.State, listeners map[string]net.Listener) {
	portState := config.PortMapState{
		Mappings: []config.PortMapping{
			{
				LocalAddress: fmt.Sprintf("127.0.0.1:%d", s.Cfg.VM.SSHPort),
				VMAddress:    "0.0.0.0:22",
				Proto:        "tcp",
				Status:       "SYSTEM",
			},
		},
	}

	for port := range listeners {
		parts := strings.Fields(port)
		if len(parts) == 1 {
			parts = append(parts, "tcp")
		}
		portState.Mappings = append(portState.Mappings, config.PortMapping{
			LocalAddress: fmt.Sprintf("127.0.0.1:%s", parts[0]),
			VMAddress:    ":::" + parts[0],
			Proto:        parts[1],
			Status:       "ACTIVE",
		})
	}

	_ = config.SavePortMappings(s.GetPortMapFile(), portState)
}

// Low overhead parser to fetch active ports inside Alpine
func getActiveVMPorts(client *ssh.Client) (map[string]bool, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Close() }()

	var buf bytes.Buffer
	session.Stdout = &buf

	// Fetch listening sockets filtering out column headers
	err = session.Run("netstat -tunl | awk '/:/{print $4, $1}'")
	if err != nil {
		return nil, err
	}

	openPorts := map[string]bool{}
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Find the last colon to correctly identify the port, supporting IPv6 (e.g. :::80)
		idx := strings.LastIndex(line, ":")
		if idx != -1 {
			portStr := line[idx+1:]
			decPort, _ := strconv.Atoi(strings.Fields(portStr)[0])
			if decPort > 0 {
				openPorts[portStr] = true
			}
		}
	}
	return openPorts, scanner.Err()
}
