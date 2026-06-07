package ssh

import (
	"bufio"
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

// StartConnForwarder bridges the Docker socket from the VM to the host and sets up automatic port-forwarding
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
		for _, l := range activeListeners {
			_ = l.Close()
		}
		_ = os.Remove(state.GetPortMapFile())
	}()

	// Initial update to show SYSTEM ports
	updatePortState(state, activeListeners)

	for ports := range StreamVMPorts(ctx, client, interval) {
		changed := false

		// Spin up dynamic forwarders for newly discovered container ports
		for id := range ports {
			// Avoid clashing with systemic default ports
			if id == "" || id == "22 tcp" || id == "68 udp" || id == "546 udp" {
				continue
			}

			parts := strings.Fields(id)
			if len(parts) < 2 || parts[1] == "udp" {
				// TODO: implement UDP tunnel
				continue
			}

			if _, exists := activeListeners[id]; !exists {
				localAddr := "localhost:" + parts[0]
				listener, err := net.Listen(parts[1], localAddr)
				if err != nil {
					fmt.Printf("[-] Failed to listen on %s %s: %v\n", parts[1], localAddr, err)
					continue
				}
				activeListeners[id] = listener
				fmt.Printf("[+] Auto-Forwarding detected %s port: %s -> VM Port %s\n", parts[1], localAddr, parts[0])
				changed = true

				dial := func() (io.ReadWriteCloser, error) {
					return client.Dial(parts[1], "localhost:"+parts[0])
				}
				go ServeListener(ctx, listener, dial)
			}
		}

		// Clean up closed ports
		for id, l := range activeListeners {
			if !ports[id] {
				fmt.Printf("[-] Stopped forwarding port: %s\n", id)
				_ = l.Close()
				delete(activeListeners, id)
				changed = true
			}
		}

		if changed {
			updatePortState(state, activeListeners)
		}
	}

	if ctx.Err() == nil {
		return fmt.Errorf("port monitoring stream closed unexpectedly")
	}
	return nil
}

// updatePortState persists the current mapping of active host listeners to the profile's port state file.
func updatePortState(s *config.State, listeners map[string]net.Listener) {
	mappings := []config.PortMapping{{
		LocalAddress: fmt.Sprintf("127.0.0.1:%d", s.Cfg.VM.SSHPort),
		VMAddress:    "0.0.0.0:22",
		Proto:        "tcp",
		Status:       "SYSTEM",
	}}

	for id := range listeners {
		parts := strings.Fields(id)
		mappings = append(mappings, config.PortMapping{
			LocalAddress: "127.0.0.1:" + parts[0],
			VMAddress:    "0.0.0.0:" + parts[0],
			Proto:        parts[1],
			Status:       "ACTIVE",
		})
	}

	_ = config.SavePortMappings(s.GetPortMapFile(), config.PortMapState{Mappings: mappings})
}

// StreamVMPorts opens a long-lived SSH session to monitor active ports inside the VM.
// It yields a map of active "port protocol" strings to the caller via a channel.
func StreamVMPorts(ctx context.Context, client *ssh.Client, interval time.Duration) <-chan map[string]bool {
	updates := make(chan map[string]bool)

	go func() {
		defer close(updates)

		session, err := client.NewSession()
		if err != nil {
			return
		}
		defer func() { _ = session.Close() }()

		stdout, err := session.StdoutPipe()
		if err != nil {
			return
		}

		// Monitoring /proc/net files
		monitorCmd := fmt.Sprintf(`while true; do
			awk 'FNR>1 && ($4 == "0A" || FILENAME ~ /udp/) {split($2, a, ":"); print a[2], (FILENAME ~ /udp/ ? "udp" : "tcp")}' /proc/net/tcp* /proc/net/udp* 2>/dev/null
			echo "---"
			sleep %d
		done`, int(interval.Seconds()))

		if err := session.Start(monitorCmd); err != nil {
			return
		}

		scanner := bufio.NewScanner(stdout)
		current := make(map[string]bool)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "---" {
				select {
				case updates <- current:
					if len(current) > 0 {
						current = make(map[string]bool)
					}
				case <-ctx.Done():
					return
				}
				continue
			}

			if f := strings.Fields(line); len(f) == 2 {
				if p, err := strconv.ParseUint(f[0], 16, 16); err == nil {
					current[fmt.Sprintf("%d %s", p, f[1])] = true
				}
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("[-] Port discovery stream error: %v\n", err)
		}
	}()

	return updates
}
