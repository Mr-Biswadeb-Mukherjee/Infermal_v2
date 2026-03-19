package dns

import (
	"context"
	"errors"
	"time"
)

const ownResolverShare = 70

func (d *DNS) resolveWithAdaptiveFallback(ctx context.Context, domain string) (bool, error) {
	customCtx, cancel := ownResolversContext(ctx)
	defer cancel()

	ok, ownErr := runResolverChain(customCtx, domain, d.primary, d.backup, d.recursive)
	if ok {
		return true, nil
	}

	if d.system != nil {
		systemCtx, cancel := systemResolverContext(ctx)
		defer cancel()

		ok, sysErr := d.system.Resolve(systemCtx, domain)
		if ok && sysErr == nil {
			return true, nil
		}
		return false, collapseResolveErrors(ownErr, sysErr)
	}

	return false, collapseResolveErrors(ownErr, nil)
}

func ownResolversContext(ctx context.Context) (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return context.WithTimeout(ctx, 1500*time.Millisecond)
	}

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return ctx, func() {}
	}

	budget := (remaining * ownResolverShare) / 100
	if budget < 200*time.Millisecond || budget >= remaining {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, budget)
}

func systemResolverContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 1500*time.Millisecond)
}

func runResolverChain(ctx context.Context, domain string, resolvers ...Resolver) (bool, error) {
	var joined error

	for _, r := range resolvers {
		if r == nil {
			continue
		}

		ok, err := r.Resolve(ctx, domain)
		if ok && err == nil {
			return true, nil
		}
		if err != nil {
			joined = errors.Join(joined, err)
		}
	}

	return false, joined
}

func collapseResolveErrors(errs ...error) error {
	joined := errors.Join(errs...)
	if joined != nil {
		return joined
	}
	return errors.New("dns: no records found")
}
