package nut

import (
	"bufio"
	"io"
	"net"
	"testing"
	"time"
)

func TestDisconnectClosesSocket(t *testing.T) {
	for _, reply := range []bool{false, true} {
		name := "unresponsive"
		if reply {
			name = "goodbye"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			conn, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			peer, err := listener.AcceptTCP()
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			closed := make(chan error, 1)
			go func() {
				reader := bufio.NewReader(peer)
				command, err := reader.ReadString('\n')
				if err != nil {
					closed <- err
					return
				}
				if command != "LOGOUT\n" {
					closed <- io.ErrUnexpectedEOF
					return
				}
				if reply {
					_, _ = io.WriteString(peer, "OK Goodbye\n")
				}
				_, err = reader.ReadByte()
				closed <- err
			}()
			client := Client{conn: conn}
			ok, err := client.Disconnect()
			if reply && (!ok || err != nil) {
				t.Fatalf("goodbye: %v %v", ok, err)
			}
			if !reply && err == nil {
				t.Fatal("expected logout timeout")
			}
			select {
			case err := <-closed:
				if err != io.EOF {
					t.Fatalf("peer did not see socket EOF: %v", err)
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatal("Disconnect returned without closing TCP socket")
			}
		})
	}
}
