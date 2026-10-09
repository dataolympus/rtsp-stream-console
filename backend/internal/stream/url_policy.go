package stream

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

var ErrRTSPURLNotAllowed = errors.New("RTSP URL is not allowed")

type IPResolver interface {
	LookupNetIP(
		ctx context.Context,
		network string,
		host string,
	) ([]netip.Addr, error)
}

type RTSPURLPolicy struct {
	Resolver             IPResolver
	AllowedHosts         map[string]struct{}
	AllowPrivateNetworks bool
}

func (p RTSPURLPolicy) Validate(
	ctx context.Context,
	rawURL string,
) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ErrRTSPURLNotAllowed
	}

	if parsed.Scheme != "rtsp" {
		return ErrRTSPURLNotAllowed
	}

	if parsed.User != nil {
		return ErrRTSPURLNotAllowed
	}

	host := strings.ToLower(
		strings.TrimSuffix(
			parsed.Hostname(),
			".",
		),
	)

	if host == "" {
		return ErrRTSPURLNotAllowed
	}

	if _, ok := p.AllowedHosts[host]; ok {
		return nil
	}

	if host == "localhost" {
		return ErrRTSPURLNotAllowed
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		if p.isAddressBlocked(addr) {
			return ErrRTSPURLNotAllowed
		}

		return nil
	}

	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}

	addresses, err := resolver.LookupNetIP(
		ctx,
		"ip",
		host,
	)
	if err != nil || len(addresses) == 0 {
		return ErrRTSPURLNotAllowed
	}

	for _, addr := range addresses {
		if p.isAddressBlocked(addr) {
			return ErrRTSPURLNotAllowed
		}
	}

	return nil
}

func (p RTSPURLPolicy) isAddressBlocked(
	addr netip.Addr,
) bool {
	if addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified() {
		return true
	}

	if addr.IsPrivate() &&
		!p.AllowPrivateNetworks {
		return true
	}

	return false
}
