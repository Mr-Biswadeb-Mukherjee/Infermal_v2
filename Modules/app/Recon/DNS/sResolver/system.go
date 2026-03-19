package sresolver

import (
	"context"
	"errors"
	"net"
	"strings"
)

// SystemResolver delegates lookups to the host OS resolver stack.
type SystemResolver struct {
	Resolver *net.Resolver
}

func NewSystem() *SystemResolver {
	return &SystemResolver{
		Resolver: net.DefaultResolver,
	}
}

func (r *SystemResolver) Resolve(ctx context.Context, domain string) (bool, error) {
	name := strings.TrimSpace(domain)
	if name == "" {
		return false, errors.New("systemresolver: empty domain")
	}

	resolver := r.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	ips, err := resolver.LookupIPAddr(ctx, name)
	if err != nil {
		return false, err
	}
	if len(ips) == 0 {
		return false, nil
	}
	return true, nil
}
