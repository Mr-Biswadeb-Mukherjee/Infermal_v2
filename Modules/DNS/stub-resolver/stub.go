package stubresolver

import (
    "context"
    "errors"
    "time"

    "github.com/miekg/dns"
)

// StubResolver performs a DNS query against ONE upstream DNS server.
// No defaults are set here — everything is provided by dns.go via options.
type StubResolver struct {
    Upstream string        // primary DNS server (must be provided)
    Retries  int           // retries per record type
    Delay    time.Duration // delay between retries
    Timeout  time.Duration // per-request timeout
}

type Option func(*StubResolver)

// ---------------------------------------------------------
// Functional options — no defaults, all must come from dns.go
// ---------------------------------------------------------

func WithUpstream(u string) Option {
    return func(r *StubResolver) {
        r.Upstream = u
    }
}

func WithRetries(n int) Option {
    return func(r *StubResolver) {
        r.Retries = n
    }
}

func WithDelay(d time.Duration) Option {
    return func(r *StubResolver) {
        r.Delay = d
    }
}

func WithTimeout(t time.Duration) Option {
    return func(r *StubResolver) {
        r.Timeout = t
    }
}

// ---------------------------------------------------------
// Constructor — NO internal defaults
// ---------------------------------------------------------

func New(opts ...Option) *StubResolver {
    r := &StubResolver{}

    for _, opt := range opts {
        opt(r)
    }

    return r
}

// ---------------------------------------------------------
// Internal DNS query (ONE upstream only)
// ---------------------------------------------------------

func (r *StubResolver) resolveOnce(ctx context.Context, fqdn string, qtype uint16) (*dns.Msg, error) {
    if r.Upstream == "" {
        return nil, errors.New("stubresolver: upstream DNS not configured")
    }

    client := &dns.Client{
        Net:            "udp",
        Timeout:        r.Timeout,
        UDPSize:        4096,
        SingleInflight: true,
    }

    msg := new(dns.Msg)
    msg.SetQuestion(fqdn, qtype)
    msg.RecursionDesired = true

    resultCh := make(chan *dns.Msg, 1)
    errCh := make(chan error, 1)

    go func() {
        resp, _, err := client.Exchange(msg, r.Upstream)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- resp
    }()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case resp := <-resultCh:
        return resp, nil
    case err := <-errCh:
        return nil, err
    }
}

// ---------------------------------------------------------
// Resolve A then AAAA (primary DNS only)
// Backup DNS is handled at dns.go level
// ---------------------------------------------------------

func (r *StubResolver) Resolve(ctx context.Context, domain string) (bool, error) {
    if domain == "" {
        return false, errors.New("stubresolver: empty domain")
    }
    if r.Upstream == "" {
        return false, errors.New("stubresolver: upstream not configured")
    }
    if r.Retries <= 0 {
        return false, errors.New("stubresolver: retries not configured")
    }
    if r.Timeout <= 0 {
        return false, errors.New("stubresolver: timeout not configured")
    }

    fqdn := dns.Fqdn(domain)
    qtypes := []uint16{dns.TypeA, dns.TypeAAAA}

    for _, qtype := range qtypes {
        for attempt := 0; attempt < r.Retries; attempt++ {

            resp, err := r.resolveOnce(ctx, fqdn, qtype)
            if err == nil && resp != nil && len(resp.Answer) > 0 {
                return true, nil
            }

            time.Sleep(r.Delay)
        }
    }

    return false, nil
}
