package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadUsesStrictRTSPDefaults(
	t *testing.T,
) {
	t.Setenv(
		"RTSP_ALLOW_PRIVATE_NETWORKS",
		"",
	)
	t.Setenv(
		"RTSP_ALLOWED_HOSTS",
		"",
	)

	cfg := Load()

	if cfg.RTSPAllowPrivateNetworks {
		t.Fatal(
			"expected private networks to be disabled by default",
		)
	}

	if len(cfg.RTSPAllowedHosts) != 0 {
		t.Fatalf(
			"expected no allowed hosts by default, got %v",
			cfg.RTSPAllowedHosts,
		)
	}
}

func TestLoadReadsRTSPPolicyConfiguration(
	t *testing.T,
) {
	t.Setenv(
		"RTSP_ALLOW_PRIVATE_NETWORKS",
		"true",
	)

	t.Setenv(
		"RTSP_ALLOWED_HOSTS",
		"mediamtx, camera.lab.local ",
	)

	cfg := Load()

	if !cfg.RTSPAllowPrivateNetworks {
		t.Fatal(
			"expected private networks to be enabled",
		)
	}

	want := []string{
		"mediamtx",
		"camera.lab.local",
	}

	if !reflect.DeepEqual(
		cfg.RTSPAllowedHosts,
		want,
	) {
		t.Fatalf(
			"expected allowed hosts %v, got %v",
			want,
			cfg.RTSPAllowedHosts,
		)
	}
}

func TestLoadUsesDefaultMaxActiveStreams(
	t *testing.T,
) {
	t.Setenv(
		"MAX_ACTIVE_STREAMS",
		"",
	)

	cfg := Load()

	if cfg.MaxActiveStreams != 4 {
		t.Fatalf(
			"expected default max active streams 4, got %d",
			cfg.MaxActiveStreams,
		)
	}
}

func TestLoadReadsMaxActiveStreams(
	t *testing.T,
) {
	t.Setenv(
		"MAX_ACTIVE_STREAMS",
		"2",
	)

	cfg := Load()

	if cfg.MaxActiveStreams != 2 {
		t.Fatalf(
			"expected max active streams 2, got %d",
			cfg.MaxActiveStreams,
		)
	}
}

func TestLoadFallsBackForInvalidMaxActiveStreams(
	t *testing.T,
) {
	t.Setenv(
		"MAX_ACTIVE_STREAMS",
		"invalid",
	)

	cfg := Load()

	if cfg.MaxActiveStreams != 4 {
		t.Fatalf(
			"expected fallback max active streams 4, got %d",
			cfg.MaxActiveStreams,
		)
	}
}

func TestLoadUsesDefaultMaxViewersPerStream(
	t *testing.T,
) {
	t.Setenv(
		"MAX_VIEWERS_PER_STREAM",
		"",
	)

	cfg := Load()

	if cfg.MaxViewersPerStream != 8 {
		t.Fatalf(
			"expected default max viewers per stream 8, got %d",
			cfg.MaxViewersPerStream,
		)
	}
}

func TestLoadReadsMaxViewersPerStream(
	t *testing.T,
) {
	t.Setenv(
		"MAX_VIEWERS_PER_STREAM",
		"4",
	)

	cfg := Load()

	if cfg.MaxViewersPerStream != 4 {
		t.Fatalf(
			"expected max viewers per stream 4, got %d",
			cfg.MaxViewersPerStream,
		)
	}
}

func TestLoadFallsBackForInvalidMaxViewersPerStream(
	t *testing.T,
) {
	t.Setenv(
		"MAX_VIEWERS_PER_STREAM",
		"0",
	)

	cfg := Load()

	if cfg.MaxViewersPerStream != 8 {
		t.Fatalf(
			"expected fallback max viewers per stream 8, got %d",
			cfg.MaxViewersPerStream,
		)
	}
}

func TestLoadUsesDefaultHTTPServerTimeouts(
	t *testing.T,
) {
	cfg := Load()

	if cfg.HTTPReadHeaderTimeout != 5*time.Second {
		t.Fatalf(
			"expected read header timeout %v, got %v",
			5*time.Second,
			cfg.HTTPReadHeaderTimeout,
		)
	}

	if cfg.HTTPIdleTimeout != 60*time.Second {
		t.Fatalf(
			"expected idle timeout %v, got %v",
			60*time.Second,
			cfg.HTTPIdleTimeout,
		)
	}
}

func TestLoadUsesDefaultRateLimitConfiguration(
	t *testing.T,
) {
	t.Setenv(
		"EXPENSIVE_REQUESTS_PER_MINUTE",
		"",
	)

	t.Setenv(
		"TRUST_PROXY_HEADERS",
		"",
	)

	cfg := Load()

	if cfg.ExpensiveRequestsPerMinute != 10 {
		t.Fatalf(
			"expected default expensive requests per minute 10, got %d",
			cfg.ExpensiveRequestsPerMinute,
		)
	}

	if cfg.TrustProxyHeaders {
		t.Fatal(
			"expected proxy headers not to be trusted by default",
		)
	}
}

func TestLoadReadsRateLimitConfiguration(
	t *testing.T,
) {
	t.Setenv(
		"EXPENSIVE_REQUESTS_PER_MINUTE",
		"6",
	)

	t.Setenv(
		"TRUST_PROXY_HEADERS",
		"true",
	)

	cfg := Load()

	if cfg.ExpensiveRequestsPerMinute != 6 {
		t.Fatalf(
			"expected expensive requests per minute 6, got %d",
			cfg.ExpensiveRequestsPerMinute,
		)
	}

	if !cfg.TrustProxyHeaders {
		t.Fatal(
			"expected proxy headers to be trusted",
		)
	}
}
