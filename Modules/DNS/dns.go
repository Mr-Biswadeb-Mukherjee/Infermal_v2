package dns

import (
	"context"
	"errors"
	"time"

	stubresolver "github.com/official-biswadeb941/Infermal_v2/Modules/DNS/stub-resolver"
)

// Resolver is the minimal interface any resolver must implement.
type Resolver interface {
	Resolve(ctx context.Context, domain string) (bool, error)
}

// DNS orchestrator supports a single primary resolver, an optional backup resolver,
// and an optional recursive resolver (for future use). The recursive resolver is
// nil by default and only used when both primary and backup fail.
type DNS struct {
	primary   Resolver
	backup    Resolver
	recursive Resolver // optional; may be nil
}

// Config contains DNS settings (mapped from setting.conf via config.go).
// Keep this small and explicit so dns.go can build the appropriate resolvers.
type Config struct {
	Upstream  string // primary upstream (upstream_dns)
	Backup    string // backup upstream (backup_dns), optional
	Retries   int    // number of retries per record type (dns_retries)
	TimeoutMS int64  // per-request timeout in milliseconds (dns_timeout_ms)
	DelayMS   int64  // retry delay in milliseconds (optional; sensible default if 0)
}

// New constructs the DNS orchestrator using the stub resolver(s).
// It does NOT enable recursive resolution — that must be attached explicitly.
func New(cfg Config) *DNS {
	// Convert time values
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	delay := time.Duration(cfg.DelayMS) * time.Millisecond
	if delay <= 0 {
		// sensible default delay between retries; you can change this via Config
		delay = 50 * time.Millisecond
	}

	// Primary resolver MUST be configured — if not, create a stub that will return error on use.
	var primary Resolver
	if cfg.Upstream != "" {
		primary = stubresolver.New(
			stubresolver.WithUpstream(cfg.Upstream),
			stubresolver.WithRetries(cfg.Retries),
			stubresolver.WithTimeout(timeout),
			stubresolver.WithDelay(delay),
		)
	}

	// Backup resolver is optional and only created if a backup upstream is provided.
	var backup Resolver
	if cfg.Backup != "" {
		backup = stubresolver.New(
			stubresolver.WithUpstream(cfg.Backup),
			stubresolver.WithRetries(cfg.Retries),
			stubresolver.WithTimeout(timeout),
			stubresolver.WithDelay(delay),
		)
	}

	return &DNS{
		primary: primary,
		backup:  backup,
		// recursive remains nil until explicitly attached via AttachRecursive
	}
}

// AttachRecursive attaches a recursive resolver to the orchestrator.
// The recursive resolver is optional and will only be called if both primary
// and backup fail. This keeps current behavior unchanged until you enable it.
func (d *DNS) AttachRecursive(r Resolver) {
	d.recursive = r
}

// Resolve attempts resolution using the following order:
//  1) primary resolver (always)
//  2) backup resolver (only if primary fails or returns no records)
//  3) recursive resolver (only if both primary and backup fail and recursive is set)
//
// The backup and recursive resolvers are invoked only on failure of the previous stage,
// guaranteeing zero performance hit from optional resolvers when primary is healthy.
func (d *DNS) Resolve(ctx context.Context, domain string) (bool, error) {
	// Validate primary configured
	if d.primary == nil {
		return false, errors.New("dns: primary resolver not configured")
	}

	// First, try primary and return immediately on success.
	ok, err := d.primary.Resolve(ctx, domain)
	if err == nil && ok {
		return true, nil
	}

	// If primary returned success==false (no records) but no error, still try backup.
	// If primary returned an error, also try backup if available.
	if d.backup != nil {
		ok2, err2 := d.backup.Resolve(ctx, domain)
		if err2 == nil && ok2 {
			return true, nil
		}
		// If backup succeeded/failed, return that result (or fall-through to recursive).
		// We choose to continue to recursive only if backup did not succeed.
	}

	// If recursive resolver is attached, only now call it.
	if d.recursive != nil {
		return d.recursive.Resolve(ctx, domain)
	}

	// Nothing succeeded — return primary error if present (prefer meaningful error), else generic.
	if err != nil {
		return false, err
	}
	return false, errors.New("dns: no records found")
}

// SwapPrimary allows swapping the primary resolver at runtime.
func (d *DNS) SwapPrimary(r Resolver) {
	d.primary = r
}

// SwapBackup allows swapping the backup resolver at runtime.
func (d *DNS) SwapBackup(r Resolver) {
	d.backup = r
}
