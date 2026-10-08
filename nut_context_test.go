package nut

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

func TestConnectRejectsFailedHandshake(t *testing.T) {
	for _, failAt := range []string{"VER", "NETVER"} {
		t.Run(failAt, func(t *testing.T) {
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
				for _, command := range []string{"VER", "NETVER"} {
					line, err := reader.ReadString('\n')
					if err != nil || line != command+"\n" {
						closed <- io.ErrUnexpectedEOF
						return
					}
					if command == failAt {
						_, _ = io.WriteString(peer, "ERR UNKNOWN-COMMAND\n")
						_, err = reader.ReadByte()
						closed <- err
						return
					}
					_, _ = io.WriteString(peer, "2.8.0\n")
				}
			}()
			client, err := Connect("127.0.0.1", listener.Addr().(*net.TCPAddr).Port)
			defer client.Close()
			if err == nil {
				t.Fatal("Connect accepted a failed handshake")
			}
			select {
			case err := <-closed:
				if err != io.EOF {
					t.Fatalf("failed handshake did not close the socket: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("failed handshake left the socket open")
			}
		})
	}
}
