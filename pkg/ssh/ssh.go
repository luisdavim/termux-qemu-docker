package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/retry"
)

func getSSHHostKey(s *config.State, key ssh.PublicKey) ([]byte, error) {
	if len(s.SSHHostKey) != 0 {
		return s.SSHHostKey, nil
	}

	keyPath := s.GetSSHHostKeyFile()
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		keyData := ssh.MarshalAuthorizedKey(key)
		err := os.WriteFile(keyPath, keyData, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to save host key: %w", err)
		}
		s.SSHHostKey = keyData
		return keyData, nil
	}

	var err error
	s.SSHHostKey, err = os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return s.SSHHostKey, nil
}

func hostKeyCallback(s *config.State) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		trustedKeyBytes, err := getSSHHostKey(s, key)
		if err != nil {
			return err
		}

		trustedKey, _, _, _, err := ssh.ParseAuthorizedKey(trustedKeyBytes)
		if err != nil {
			return err
		}

		if ssh.FingerprintSHA256(key) != ssh.FingerprintSHA256(trustedKey) {
			return fmt.Errorf("SECURITY ALERT: Host key mismatch for %s", s.Profile)
		}

		return nil
	}
}

func GetClient(ctx context.Context, s *config.State) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	keyPath := GetKeyPath(s.HomeDir)
	if keyBytes, err := os.ReadFile(keyPath); err == nil {
		if signer, err := ssh.ParsePrivateKey(keyBytes); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	authMethods = append(authMethods, ssh.Password(s.Cfg.VM.SSHPassword))

	sshConfig := &ssh.ClientConfig{
		User:            s.Cfg.VM.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback(s),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.Cfg.VM.SSHPort)
	var client *ssh.Client
	var err error

	err = retry.WithTimeout(ctx, 5*time.Minute, time.Second, 5*time.Second, func() error {
		client, err = ssh.Dial("tcp", addr, sshConfig)
		if err == nil {
			// Start keep-alive goroutine
			go func() {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						_, _, err = client.SendRequest("keepalive@openssh.com", true, nil)
						if err != nil {
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}()
			return nil
		}
		fmt.Printf("... waiting for VM network: %v\n", err)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("core engine communication failed after several attempts: %w", err)
	}

	return client, nil
}

func Shell(s *config.State) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := GetClient(ctx, s)
	if err != nil {
		return fmt.Errorf("failed to ge SSH client: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{}
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 80, 40
	}

	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}

	if err := session.RequestPty(termType, height, width, modes); err != nil {
		return fmt.Errorf("request for pseudo-terminal failed: %w", err)
	}

	// Handle window resizing
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer func() {
		signal.Stop(sigwinch)
		close(sigwinch)
	}()

	go func() {
		for range sigwinch {
			width, height, err := term.GetSize(fd)
			if err == nil {
				_ = session.WindowChange(height, width)
			}
		}
	}()

	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("session failed during execution: %w", err)
	}

	return nil
}

func RunCommand(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()
	return session.Run(cmd)
}
