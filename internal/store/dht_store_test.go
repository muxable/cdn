package store

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDHTStore_Simple(t *testing.T) {
	s1conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := NewDHTStore(context.Background(), s1conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	
	s2conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewDHTStore(context.Background(), s2conn, s1.Addr())
	if err != nil {
		t.Fatal(err)
	}

	if err := s1.Put("moo", "cow"); err != nil {
		t.Fatal(err)
	}

	got, err := s2.Get("moo")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cow" {
		t.Errorf("got %q, want %q", got, "cow")
	}

	got, err = s1.Get("moo")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cow" {
		t.Errorf("got %q, want %q", got, "cow")
	}
}

func TestDHTStore_RouteThroughThird(t *testing.T) {
	s1conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := NewDHTStore(context.Background(), s1conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	
	s2conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewDHTStore(context.Background(), s2conn, s1.Addr())
	if err != nil {
		t.Fatal(err)
	}
	
	s3conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s3, err := NewDHTStore(context.Background(), s3conn, s2.Addr())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	nodes, err := s1.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}

	nodes, err = s2.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}

	nodes, err = s3.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}
	
	time.Sleep(100 * time.Millisecond)

	// write to s3
	if err := s3.Put("moo", "sheep"); err != nil {
		t.Fatal(err)
	}
	
	time.Sleep(100 * time.Millisecond)

	// read from s2
	got, err := s2.Get("moo")
	if err != nil {
		t.Fatal(err)
	}

	if got != "sheep" {
		t.Errorf("got %q, want %q", got, "sheep")
	}

	time.Sleep(100 * time.Millisecond)

	// write to s1
	if err := s1.Put("baa", "cow"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	
	// read from s3
	got, err = s3.Get("baa")
	if err != nil {
		t.Fatal(err)
	}

	if got != "cow" {
		t.Errorf("got %q, want %q", got, "cow")
	}

	s1.Close()
	s2.Close()
	s3.Close()
}

func TestDHTStore_PublisherDestroyed(t *testing.T) {
	s1conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := NewDHTStore(context.Background(), s1conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	
	s2conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := NewDHTStore(context.Background(), s2conn, s1.Addr())
	if err != nil {
		t.Fatal(err)
	}
	
	s3conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s3, err := NewDHTStore(context.Background(), s3conn, s2.Addr())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	nodes, err := s1.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}

	nodes, err = s2.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}

	nodes, err = s3.TraversalStartingNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}
	
	time.Sleep(100 * time.Millisecond)

	// write to s3
	if err := s3.Put("moo", "sheep"); err != nil {
		t.Fatal(err)
	}
	
	time.Sleep(100 * time.Millisecond)

	s3.Close()

	// read from s1
	got, err := s1.Get("moo")
	if err != nil {
		t.Fatal(err)
	}

	if got != "sheep" {
		t.Errorf("got %q, want %q", got, "sheep")
	}

	s1.Close()
	s2.Close()
}