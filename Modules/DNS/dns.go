package dns

import (
	"context"
	"errors"
	"time"

	stubresolver "github.com/official-biswadeb941/Infermal_v2/Modules/DNS/stub-resolver"
)

//
// ===================================================
//               PUBLIC STABLE API LAYER
// ===================================================
//
// These functions MUST NEVER change. They form the
// external contract that app.go and other upper-level
// orchestrators depend on.
//

// globalDNS holds the shared instance created via InitDNS.
var globalDNS *DNS

// InitDNS initializes the DNS subsystem using the provided config.
// This function signature must remain unchanged forever.
func InitDNS(cfg Config) (*DNS, error) {
	d := New(cfg)

	if d.primary == nil {
		return nil, errors.New("dns: upstream resolver not set")
	}

	globalDNS = d
	return d, nil
}

// ResolveDomain is the simple, top-level resolver used by app.go.
// Even if internal logic changes, this must remain compatible.
func ResolveDomain(domain string) (bool, error) {
	if globalDNS == nil {
		return false, errors.New("dns: module not initialized (InitDNS not called)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return globalDNS.Resolve(ctx, domain)
}

// ResolveWithContext allows advanced callers to provide their own context.
// Signature MUST remain unchanged.
func ResolveWithContext(ctx context.Context, domain string) (bool, error) {
	if globalDNS == nil {
		return false, errors.New("dns: module not initialized (InitDNS not called)")
	}
	return globalDNS.Resolve(ctx, domain)
}

// Health returns nil when DNS is ready.
// Signature MUST remain unchanged.
func Health() error {
	if globalDNS == nil {
		return errors.New("dns: not initialized")
	}
	if globalDNS.primary == nil {
		return errors.New("dns: primary resolver missing")
	}
	return nil
}

//
// ===================================================
//                   CONSTRUCTOR
// ===================================================
//

func New(cfg Config) *DNS {
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	delay := time.Duration(cfg.DelayMS) * time.Millisecond
	if delay <= 0 {
		delay = 50 * time.Millisecond
	}

	var primary Resolver
	if cfg.Upstream != "" {
		primary = stubresolver.New(
			stubresolver.WithUpstream(cfg.Upstream),
			stubresolver.WithRetries(cfg.Retries),
			stubresolver.WithTimeout(timeout),
			stubresolver.WithDelay(delay),
		)
	}

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
		// recursive stays nil until AttachRecursive is called
	}
}

//
// ===================================================
//                   RESOLUTION
// ===================================================
//

// Resolve orchestrates domain evaluation through:
// 1) Optional cache
// 2) Primary resolver
// 3) Backup resolver
// 4) Recursive resolver
func (d *DNS) Resolve(ctx context.Context, domain string) (bool, error) {

	//
	// 0) CACHE LOOKUP (non-blocking with timeout)
	//
	if d.cache != nil {
		cacheCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		val, err := d.cache.GetValue(cacheCtx, "dns:"+domain)
		cancel()

		if err == nil {
			switch val {
			case "1":
				return true, nil
			case "0":
				return false, nil
			}
		}
	}

	//
	// Prepare strict per-domain timeout
	//
	domainCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	//
	// 1) PRIMARY RESOLVER
	//
	if d.primary == nil {
		return false, errors.New("dns: primary resolver not configured")
	}

	ok, err := d.primary.Resolve(domainCtx, domain)
	if err == nil && ok {
		d.asyncCacheWrite(domain, "1", 48*time.Hour)
		return true, nil
	}

	//
	// 2) BACKUP RESOLVER (optional)
	//
	if d.backup != nil {
		ok2, err2 := d.backup.Resolve(domainCtx, domain)
		if err2 == nil && ok2 {
			d.asyncCacheWrite(domain, "1", 48*time.Hour)
			return true, nil
		}
	}

	//
	// 3) RECURSIVE RESOLVER (optional)
	//
	if d.recursive != nil {
		ok3, err3 := d.recursive.Resolve(domainCtx, domain)
		if err3 == nil && ok3 {
			d.asyncCacheWrite(domain, "1", 48*time.Hour)
			return true, nil
		}
	}

	//
	// ALL FAILED → cache negative result
	//
	d.asyncCacheWrite(domain, "0", 12*time.Hour)

	if err != nil {
		return false, err
	}
	return false, errors.New("dns: no records found")
}

//
// ===================================================
//           INTERNAL: ASYNC CACHE WRITER
// ===================================================
//

func (d *DNS) asyncCacheWrite(domain string, val string, ttl time.Duration) {
	if d.cache == nil {
		return
	}

	go func() {
		cctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		_ = d.cache.SetValue(cctx, "dns:"+domain, val, ttl)
		cancel()
	}()
}
