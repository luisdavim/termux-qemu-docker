package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32*1024)
		return &b
	},
}

// Proxy bidirectionally copies data between two connections.
// It returns when one side is closed or the context is canceled.
func Proxy(ctx context.Context, conn1, conn2 io.ReadWriteCloser) {
	defer func() {
		_ = conn1.Close()
		_ = conn2.Close()
	}()

	done := make(chan struct{}, 2)

	copyConn := func(dst, src io.ReadWriteCloser) {
		bufPtr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufPtr)

		_, _ = io.CopyBuffer(dst, src, *bufPtr)
		done <- struct{}{}
	}

	go copyConn(conn1, conn2)
	go copyConn(conn2, conn1)

	select {
	case <-done:
	case <-ctx.Done():
	}
}

// DialFunc is a function that establishes a connection to a remote endpoint.
type DialFunc func() (io.ReadWriteCloser, error)

// ServeListener accepts connections from l and proxies them using the provided dialer.
func ServeListener(ctx context.Context, l net.Listener, dial DialFunc) {
	defer func() { _ = l.Close() }()

	for {
		localConn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				// Only log if it's not a temporary error or context cancellation
				return
			}
		}

		go func() {
			remoteConn, err := dial()
			if err != nil {
				fmt.Printf("[-] Failed to establish remote connection: %v\n", err)
				_ = localConn.Close()
				return
			}
			Proxy(ctx, localConn, remoteConn)
		}()
	}
}
