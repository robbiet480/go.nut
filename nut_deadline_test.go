package nut

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWriteHasDeadlineWithoutContext(t *testing.T) {
	conn, peer := net.Pipe()
	defer peer.Close()
	client := Client{conn: conn}
	defer client.Close()
	result := make(chan error, 1)
	go func() { _, err := client.SendCommand("VER"); result <- err }()
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("blocked write succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("blocked write has no deadline")
	}
}

func TestReadUsesContextDeadline(t *testing.T) {
	conn, peer := net.Pipe()
	defer peer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	client := Client{conn: conn, ctx: ctx}
	defer client.Close()
	start := time.Now()
	_, err := client.ReadResponse("", false)
	if err == nil {
		t.Fatal("blocked read succeeded")
	}
	if time.Since(start) > time.Second {
		t.Fatal("read ignored context deadline")
	}
}
