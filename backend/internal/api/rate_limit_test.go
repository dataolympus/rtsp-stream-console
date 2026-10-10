package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFixedWindowLimiterEnforcesLimitPerKey(
	t *testing.T,
) {
	limiter := newFixedWindowLimiter(
		2,
		time.Minute,
	)

	if allowed, _ := limiter.allow("client-a"); !allowed {
		t.Fatal(
			"expected first request to be allowed",
		)
	}

	if allowed, _ := limiter.allow("client-a"); !allowed {
		t.Fatal(
			"expected second request to be allowed",
		)
	}

	if allowed, _ := limiter.allow("client-a"); allowed {
		t.Fatal(
			"expected third request to be rejected",
		)
	}

	if allowed, _ := limiter.allow("client-b"); !allowed {
		t.Fatal(
			"expected different client to have independent quota",
		)
	}
}

func TestFixedWindowLimiterResetsAfterWindow(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.October,
		10,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	limiter := newFixedWindowLimiter(
		1,
		time.Minute,
	)

	limiter.now = func() time.Time {
		return now
	}

	if allowed, _ := limiter.allow("client-a"); !allowed {
		t.Fatal(
			"expected first request to be allowed",
		)
	}

	if allowed, _ := limiter.allow("client-a"); allowed {
		t.Fatal(
			"expected request inside window to be rejected",
		)
	}

	now = now.Add(time.Minute)

	if allowed, _ := limiter.allow("client-a"); !allowed {
		t.Fatal(
			"expected request after window reset to be allowed",
		)
	}
}

func TestClientKeyUsesRemoteAddressByDefault(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		nil,
	)

	request.RemoteAddr = "203.0.113.20:54321"

	request.Header.Set(
		"X-Forwarded-For",
		"198.51.100.50",
	)

	got := clientKey(
		request,
		false,
	)

	if got != "203.0.113.20" {
		t.Fatalf(
			"expected remote address client key, got %q",
			got,
		)
	}
}

func TestClientKeyUsesForwardedAddressWhenTrusted(
	t *testing.T,
) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		nil,
	)

	request.RemoteAddr = "172.20.0.2:54321"

	request.Header.Set(
		"X-Forwarded-For",
		"198.51.100.50, 172.20.0.1",
	)

	got := clientKey(
		request,
		true,
	)

	if got != "198.51.100.50" {
		t.Fatalf(
			"expected forwarded client key, got %q",
			got,
		)
	}
}

func TestRateLimitMiddlewareRejectsRequestAfterLimit(
	t *testing.T,
) {
	limiter := newFixedWindowLimiter(
		1,
		time.Minute,
	)

	calls := 0

	next := http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			calls++

			w.WriteHeader(
				http.StatusNoContent,
			)
		},
	)

	handler := rateLimit(
		limiter,
		false,
		next,
	)

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		nil,
	)
	firstRequest.RemoteAddr =
		"203.0.113.20:50000"

	firstResponse :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		firstResponse,
		firstRequest,
	)

	if firstResponse.Code !=
		http.StatusNoContent {
		t.Fatalf(
			"expected first request status %d, got %d",
			http.StatusNoContent,
			firstResponse.Code,
		)
	}

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/streams",
		nil,
	)
	secondRequest.RemoteAddr =
		"203.0.113.20:50001"

	secondResponse :=
		httptest.NewRecorder()

	handler.ServeHTTP(
		secondResponse,
		secondRequest,
	)

	if secondResponse.Code !=
		http.StatusTooManyRequests {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusTooManyRequests,
			secondResponse.Code,
		)
	}

	if calls != 1 {
		t.Fatalf(
			"expected downstream handler to be called once, got %d",
			calls,
		)
	}

	if secondResponse.Header().
		Get("Retry-After") == "" {
		t.Fatal(
			"expected Retry-After header",
		)
	}

	if !strings.Contains(
		secondResponse.Body.String(),
		"rate limit exceeded",
	) {
		t.Fatalf(
			"unexpected response body: %s",
			secondResponse.Body.String(),
		)
	}
}

func TestFixedWindowLimiterPrunesExpiredEntries(
	t *testing.T,
) {
	now := time.Date(
		2026,
		time.October,
		10,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	limiter := newFixedWindowLimiter(
		1,
		time.Minute,
	)

	limiter.now = func() time.Time {
		return now
	}

	if allowed, _ :=
		limiter.allow("client-a"); !allowed {
		t.Fatal(
			"expected client-a to be allowed",
		)
	}

	if allowed, _ :=
		limiter.allow("client-b"); !allowed {
		t.Fatal(
			"expected client-b to be allowed",
		)
	}

	if got := len(limiter.entries); got != 2 {
		t.Fatalf(
			"expected 2 limiter entries, got %d",
			got,
		)
	}

	now = now.Add(time.Minute)

	if allowed, _ :=
		limiter.allow("client-c"); !allowed {
		t.Fatal(
			"expected client-c to be allowed",
		)
	}

	if _, exists :=
		limiter.entries["client-a"]; exists {
		t.Fatal(
			"expected expired client-a entry to be removed",
		)
	}

	if _, exists :=
		limiter.entries["client-b"]; exists {
		t.Fatal(
			"expected expired client-b entry to be removed",
		)
	}

	if got := len(limiter.entries); got != 1 {
		t.Fatalf(
			"expected only current client entry, got %d",
			got,
		)
	}
}
