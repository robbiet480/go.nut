package nut

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestConnectContextCancelsHandshakeAndClosesSocket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	closed := make(chan error, 1)
	go func() {
		peer, err := listener.Accept()
		if err != nil {
			closed <- err
			return
		}
		defer peer.Close()
		_ = peer.SetDeadline(time.Now().Add(3 * time.Second))
		reader := bufio.NewReader(peer)
		_, err = reader.ReadString('\n')
		if err == nil {
			_, err = reader.ReadByte()
		}
		closed <- err
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = ConnectContext(ctx, "127.0.0.1", listener.Addr().(*net.TCPAddr).Port)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline error, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("handshake exceeded overall budget")
	}
	select {
	case err := <-closed:
		if err != io.EOF {
			t.Fatalf("peer did not observe close: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled handshake left socket open")
	}
}

func TestConnectContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ConnectContext(ctx, "127.0.0.1", 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want cancellation, got %v", err)
	}
}

func TestCancellationClosesActiveIO(t *testing.T) {
	for _, operation := range []string{"read", "write", "logout", "slow"} {
		operation := operation
		t.Run(operation, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			ready := make(chan struct{})
			closed := make(chan error, 1)
			var ctx context.Context
			var cancel context.CancelFunc
			if operation == "slow" {
				ctx, cancel = context.WithTimeout(context.Background(), 250*time.Millisecond)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			go func() {
				peer, err := listener.Accept()
				if err != nil {
					closed <- err
					return
				}
				defer peer.Close()
				if operation == "write" {
					if err := peer.(*net.TCPConn).SetReadBuffer(64 << 10); err != nil {
						closed <- err
						return
					}
				}
				_ = peer.SetDeadline(time.Now().Add(3 * time.Second))
				reader := bufio.NewReader(peer)
				for _, command := range []string{"VER\n", "NETVER\n"} {
					line, err := reader.ReadString('\n')
					if err != nil || line != command {
						closed <- io.ErrUnexpectedEOF
						return
					}
					_, _ = io.WriteString(peer, "2.8.0\n")
				}
				if operation == "write" {
					_, err = reader.ReadByte()
				} else {
					_, err = reader.ReadString('\n')
				}
				if err != nil {
					closed <- err
					return
				}
				close(ready)
				if operation == "slow" {
					for ctx.Err() == nil {
						if _, err := io.WriteString(peer, "VAR demo input.voltage \"238\"\n"); err != nil {
							break
						}
						time.Sleep(25 * time.Millisecond)
					}
				} else {
					<-ctx.Done()
				}
				_, err = io.Copy(io.Discard, reader)
				closed <- err
			}()
			client, err := ConnectContext(ctx, "127.0.0.1", listener.Addr().(*net.TCPAddr).Port)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if operation == "write" {
				if err := client.conn.(*net.TCPConn).SetWriteBuffer(1024); err != nil {
					t.Fatal(err)
				}
			}
			result := make(chan error, 1)
			go func() {
				var err error
				switch operation {
				case "write":
					_, err = client.SendCommand(strings.Repeat("x", 1<<20))
				case "logout":
					_, err = client.Disconnect()
				case "slow":
					_, err = client.SendCommand("LIST VAR demo")
				default:
					_, err = client.SendCommand("VER")
				}
				result <- err
			}()
			select {
			case <-ready:
			case <-time.After(time.Second):
				t.Fatal("operation did not reach peer")
			}
			if operation != "slow" {
				cancel()
			}
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("cancelled operation succeeded")
				}
			case <-time.After(time.Second):
				t.Fatal("cancelled I/O did not stop")
			}
			select {
			case err := <-closed:
				if err != nil {
					t.Fatalf("peer did not drain to EOF: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatalf("peer did not reach EOF; local socket deadline check: %v", client.conn.SetDeadline(time.Now()))
			}
			if err := client.conn.SetDeadline(time.Now()); !errors.Is(err, net.ErrClosed) {
				t.Fatalf("context did not close the local socket: %v", err)
			}
			if err := client.Close(); err != nil {
				t.Fatal(err)
			}
			if err := client.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
