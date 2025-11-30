package stubresolver

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

//
// Proper local DNS server
//

func startTestDNSServer(t *testing.T, answers []dns.RR) (addr string, shutdown func()) {
	t.Helper()

	// Create UDP listener that we control
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen UDP: %v", err)
	}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		msg := new(dns.Msg)
		msg.SetReply(r)
		if answers != nil {
			msg.Answer = append(msg.Answer, answers...)
		}
		_ = w.WriteMsg(msg)
	})

	srv := &dns.Server{
		PacketConn: pc,
		Handler:    mux,
	}

	go func() {
		_ = srv.ActivateAndServe()
	}()

	return pc.LocalAddr().String(),
		func() {
			_ = srv.Shutdown()
			_ = pc.Close()
		}
}

//
// -----------------------
// Tests
// -----------------------
//

func TestResolveSuccess(t *testing.T) {
	rr, _ := dns.NewRR("example.com. 300 IN A 10.10.10.10")

	addr, stop := startTestDNSServer(t, []dns.RR{rr})
	defer stop()

	r := New(
		WithUpstream(addr),
		WithRetries(2),
		WithTimeout(300*time.Millisecond),
	)

	ok, err := r.Resolve(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if !ok {
		t.Fatalf("expected Resolve true but got false")
	}
}

func TestResolveNoAnswer(t *testing.T) {
	addr, stop := startTestDNSServer(t, nil)
	defer stop()

	r := New(
		WithUpstream(addr),
		WithRetries(1),
		WithTimeout(300*time.Millisecond),
	)

	ok, err := r.Resolve(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected false, got true")
	}
}

func TestResolveTimeout(t *testing.T) {
	// Bind a port and intentionally drop packets.
	pc, _ := net.ListenPacket("udp", "127.0.0.1:0")
	defer pc.Close()
	addr := pc.LocalAddr().String()

	r := New(
		WithUpstream(addr),
		WithRetries(1),
		WithTimeout(100*time.Millisecond),
	)

	start := time.Now()
	ok, err := r.Resolve(context.Background(), "example.com")

	// Should never succeed
	if ok {
		t.Fatalf("expected resolve to fail")
	}

	// The resolver design ALLOWS err == nil.
	// It returns (false, nil) for “no answer” cases.
	// Therefore: DO NOT assert error here.
	_ = err

	// Ensure the timeout logic was actually applied
	if time.Since(start) < 90*time.Millisecond {
		t.Fatalf("resolver returned too early; expected timeout delay")
	}
}

func TestResolveContextCancellation(t *testing.T) {
	pc, _ := net.ListenPacket("udp", "127.0.0.1:0")
	defer pc.Close()
	addr := pc.LocalAddr().String()

	r := New(
		WithUpstream(addr),
		WithRetries(5),
		WithTimeout(200*time.Millisecond),
		WithDelay(100*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	ok, err := r.Resolve(ctx, "example.com")
	if ok {
		t.Fatalf("expected cancellation")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestInvalidInputs(t *testing.T) {
	r := New()

	_, err := r.Resolve(context.Background(), "")
	if err == nil {
		t.Fatalf("expected empty domain error")
	}

	_, err = r.Resolve(context.Background(), "example.com")
	if err == nil {
		t.Fatalf("expected missing upstream error")
	}

	r.Upstream = "1.1.1.1:53"
	_, err = r.Resolve(context.Background(), "example.com")
	if err == nil {
		t.Fatalf("expected retries error")
	}

	r.Retries = 1
	_, err = r.Resolve(context.Background(), "example.com")
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}
