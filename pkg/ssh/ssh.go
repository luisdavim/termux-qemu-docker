package ssh

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func GetClient(ctx context.Context, c *config.Config, homeDir string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	keyPath := GetKeyPath(homeDir)
	if keyBytes, err := os.ReadFile(keyPath); err == nil {
		if signer, err := ssh.ParsePrivateKey(keyBytes); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}
	authMethods = append(authMethods, ssh.Password(c.VM.SSHPassword))

	sshConfig := &ssh.ClientConfig{
		User:            c.VM.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("127.0.0.1:%d", c.VM.SSHPort)
	var client *ssh.Client
	var err error

	for i := range 60 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			client, err = ssh.Dial("tcp", addr, sshConfig)
			if err == nil {
				return client, nil
			}
			fmt.Printf("... waiting for VM network (attempt %d/60): %v\n", i+1, err)
			time.Sleep(5 * time.Second)
		}
	}

	return nil, fmt.Errorf("core engine communication failed after 60 attempts: %w", err)
}

func Shell(s *config.State) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := GetClient(ctx, s.Cfg, s.HomeDir)
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
	defer term.Restore(fd, oldState)

	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 80, 40
	}

	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("request for pseudo-terminal failed: %w", err)
	}

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
