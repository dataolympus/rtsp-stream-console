package stream

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

type fakeIPResolver struct {
	addresses map[string][]netip.Addr
}

func (r fakeIPResolver) LookupNetIP(
	_ context.Context,
	_ string,
	host string,
) ([]netip.Addr, error) {
	return r.addresses[host], nil
}

func TestRTSPURLPolicy(t *testing.T) {
	t.Parallel()

	resolver := fakeIPResolver{
		addresses: map[string][]netip.Addr{
			"camera.example.com": {
				netip.MustParseAddr("1.1.1.1"),
			},
			"internal.example.com": {
				netip.MustParseAddr("10.0.0.10"),
			},
		},
	}

	policy := RTSPURLPolicy{
		Resolver: resolver,
		AllowedHosts: map[string]struct{}{
			"mediamtx": {},
		},
	}

	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{
			name:    "allows public RTSP host",
			rawURL:  "rtsp://camera.example.com/live",
			wantErr: false,
		},
		{
			name:    "allows explicitly allowlisted host",
			rawURL:  "rtsp://mediamtx:8554/camera-1",
			wantErr: false,
		},
		{
			name:    "rejects non RTSP scheme",
			rawURL:  "http://camera.example.com/live",
			wantErr: true,
		},
		{
			name:    "rejects missing host",
			rawURL:  "rtsp:///camera-1",
			wantErr: true,
		},
		{
			name:    "rejects localhost",
			rawURL:  "rtsp://localhost:8554/camera-1",
			wantErr: true,
		},
		{
			name:    "rejects IPv4 loopback",
			rawURL:  "rtsp://127.0.0.1:8554/camera-1",
			wantErr: true,
		},
		{
			name:    "rejects IPv6 loopback",
			rawURL:  "rtsp://[::1]:8554/camera-1",
			wantErr: true,
		},
		{
			name:    "rejects private IPv4",
			rawURL:  "rtsp://10.0.0.10/live",
			wantErr: true,
		},
		{
			name:    "rejects DNS host resolving private",
			rawURL:  "rtsp://internal.example.com/live",
			wantErr: true,
		},
		{
			name:    "rejects link local metadata address",
			rawURL:  "rtsp://169.254.169.254/latest",
			wantErr: true,
		},
		{
			name:    "rejects embedded credentials",
			rawURL:  "rtsp://admin:secret@camera.example.com/live",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(
				context.Background(),
				tt.rawURL,
			)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf(
					"expected URL to be allowed, got %v",
					err,
				)
			}
		})
	}
}

func TestRTSPURLPolicyAllowsPrivateAddressWhenExplicitlyEnabled(
	t *testing.T,
) {
	t.Parallel()

	policy := RTSPURLPolicy{
		AllowPrivateNetworks: true,
	}

	err := policy.Validate(
		context.Background(),
		"rtsp://192.168.1.20:8554/camera-1",
	)

	if err != nil {
		t.Fatalf(
			"expected private RTSP address to be allowed, got %v",
			err,
		)
	}
}

func TestRTSPURLPolicyStillRejectsLoopbackWhenPrivateNetworksEnabled(
	t *testing.T,
) {
	t.Parallel()

	policy := RTSPURLPolicy{
		AllowPrivateNetworks: true,
	}

	err := policy.Validate(
		context.Background(),
		"rtsp://127.0.0.1:8554/camera-1",
	)

	if !errors.Is(
		err,
		ErrRTSPURLNotAllowed,
	) {
		t.Fatalf(
			"expected loopback to remain blocked, got %v",
			err,
		)
	}
}
