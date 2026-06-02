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

// SocketTunnel represents the active Unix socket forwarding instance.
type SocketTunnel struct {
	client           *ssh.Client // An already established SSH client connection
	localSocketPath  string      // Path where the local Unix socket will be created
	remoteSocketPath string      // Path to the Unix socket on the remote server
	listener         net.Listener
}

// NewSocketTunnel initializes a new Tunneluration.
func NewSocketTunnel(client *ssh.Client, localSocketPath, remoteSocketPath string) (*SocketTunnel, error) {
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

	return &SocketTunnel{
		client:           client,
		localSocketPath:  localSocketPath,
		remoteSocketPath: remoteSocketPath,
	}, nil
}

// Start starts the local Unix socket listener and begins proxying traffic.
// It will run until the provided context is canceled.
func (t *SocketTunnel) Start(ctx context.Context) error {
	// Clean up any stale sockets from previous runs
	_ = os.Remove(t.localSocketPath)

	listener, err := net.Listen("unix", t.localSocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on local socket: %w", err)
	}
	t.listener = listener

	// Ensure cleanup happens whenever we exit
	defer func() { _ = os.Remove(t.localSocketPath) }()

	dial := func() (io.ReadWriteCloser, error) {
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
			return nil, err
		}
		go ssh.DiscardRequests(requests)
		return channel, nil
	}

	ServeListener(ctx, t.listener, dial)
	return nil
}
