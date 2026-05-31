package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Tunnel represents the active Unix socket forwarding instance.
type Tunnel struct {
	client           *ssh.Client // An already established SSH client connection
	localSocketPath  string      // Path where the local Unix socket will be created
	remoteSocketPath string      // Path to the Unix socket on the remote server
	listener         net.Listener
}

// NewSocketTunnel initializes a new Tunneluration.
func NewSocketTunnel(client *ssh.Client, localSocketPath, remoteSocketPath string) (*Tunnel, error) {
	if client == nil {
		return nil, fmt.Errorf("ssh client cannot be nil")
	}
	if localSocketPath == "" || remoteSocketPath == "" {
		return nil, fmt.Errorf("local and remote socket paths must be specified")
	}

	// Resolve absolute path for local socket if necessary
	if !filepath.IsAbs(localSocketPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		localSocketPath = filepath.Join(cwd, localSocketPath)
	}

	return &Tunnel{
		client:           client,
		localSocketPath:  localSocketPath,
		remoteSocketPath: remoteSocketPath,
	}, nil
}

// Start starts the local Unix socket listener and begins proxying traffic.
// It will run until the provided context is canceled.
func (t *Tunnel) Start(ctx context.Context) error {
	// Clean up any stale sockets from previous runs
	_ = os.Remove(t.localSocketPath)

	listener, err := net.Listen("unix", t.localSocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on local socket: %w", err)
	}
	t.listener = listener

	// Ensure cleanup happens whenever we exit this loop
	defer t.cleanup()

	// Monitor context cancellation in a separate goroutine to close the listener
	go func() {
		<-ctx.Done()
		if t.listener != nil {
			_ = t.listener.Close()
		}
	}()

	for {
		localConn, err := t.listener.Accept()
		if err != nil {
			// Check if the accept failed because the context was canceled
			select {
			case <-ctx.Done():
				return ctx.Err() // Returns context.Canceled
			default:
				return fmt.Errorf("error accepting connection: %w", err)
			}
		}

		// Use the context to bound individual connection lifecycles if needed
		go t.handleForwarding(ctx, localConn)
	}
}

// cleanup removes the local Unix socket file from the filesystem.
func (t *Tunnel) cleanup() {
	if t.listener != nil {
		_ = t.listener.Close()
	}
	_ = os.Remove(t.localSocketPath)
}

func (t *Tunnel) handleForwarding(ctx context.Context, localConn net.Conn) {
	// Ensure connection closes when data copying finishes or context expires
	defer func() { _ = localConn.Close() }()

	// Handle early exit if context is already dead before setting up the channel
	select {
	case <-ctx.Done():
		return
	default:
	}

	// OpenSSH payload format for direct-streamlocal@openssh.com
	payload := ssh.Marshal(struct {
		SocketPath string
		Reserved   string
		Port       uint32
	}{
		SocketPath: t.remoteSocketPath,
		Reserved:   "",
		Port:       0,
	})

	channel, requests, err := t.client.OpenChannel("direct-streamlocal@openssh.com", payload)
	if err != nil {
		fmt.Printf("[-] Failed to open remote socket channel: %v\n", err)
		return
	}
	defer func() { _ = channel.Close() }()

	go ssh.DiscardRequests(requests)

	done := make(chan struct{}, 2)

	// Local -> Remote
	go func() {
		if _, err := io.Copy(channel, localConn); err != nil && err != io.EOF {
			// Silently ignore some common network errors to avoid log spam
			_ = err
		}
		done <- struct{}{}
	}()

	// Remote -> Local
	go func() {
		if _, err := io.Copy(localConn, channel); err != nil && err != io.EOF {
			// Silently ignore some common network errors to avoid log spam
			_ = err
		}
		done <- struct{}{}
	}()

	// Wait for copying to finish OR for the context to trigger a shutdown
	select {
	case <-done:
	case <-ctx.Done():
	}
}
