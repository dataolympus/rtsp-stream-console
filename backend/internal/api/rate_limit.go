package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fixedWindowEntry struct {
	count   int
	resetAt time.Time
}

type fixedWindowLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	entries     map[string]fixedWindowEntry
	now         func() time.Time
	nextCleanup time.Time
}

func clientKey(
	r *http.Request,
	trustProxyHeaders bool,
) string {
	if trustProxyHeaders {
		forwarded :=
			r.Header.Get(
				"X-Forwarded-For",
			)

		if forwarded != "" {
			first, _, _ :=
				strings.Cut(
					forwarded,
					",",
				)

			first =
				strings.TrimSpace(
					first,
				)

			if first != "" {
				return first
			}
		}
	}

	host, _, err :=
		net.SplitHostPort(
			r.RemoteAddr,
		)

	if err == nil &&
		host != "" {
		return host
	}

	return r.RemoteAddr
}

func rateLimit(
	limiter *fixedWindowLimiter,
	trustProxyHeaders bool,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			allowed, retryAfter :=
				limiter.allow(
					clientKey(
						r,
						trustProxyHeaders,
					),
				)

			if allowed {
				next.ServeHTTP(
					w,
					r,
				)

				return
			}

			seconds :=
				int64(
					(retryAfter +
						time.Second - 1) /
						time.Second,
				)

			if seconds < 1 {
				seconds = 1
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			w.Header().Set(
				"Retry-After",
				strconv.FormatInt(
					seconds,
					10,
				),
			)

			w.WriteHeader(
				http.StatusTooManyRequests,
			)

			_, _ = w.Write(
				[]byte(
					`{"error":"rate limit exceeded"}`,
				),
			)
		},
	)
}

func newFixedWindowLimiter(
	limit int,
	window time.Duration,
) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]fixedWindowEntry),
		now:     time.Now,
	}
}

func (l *fixedWindowLimiter) allow(
	key string,
) (bool, time.Duration) {
	if l.limit <= 0 {
		return true, 0
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.nextCleanup.IsZero() {
		l.nextCleanup =
			now.Add(l.window)
	} else if !now.Before(
		l.nextCleanup,
	) {
		for key, entry := range l.entries {
			if !now.Before(
				entry.resetAt,
			) {
				delete(
					l.entries,
					key,
				)
			}
		}

		l.nextCleanup =
			now.Add(l.window)
	}

	entry, exists := l.entries[key]

	if !exists ||
		!now.Before(entry.resetAt) {
		l.entries[key] = fixedWindowEntry{
			count:   1,
			resetAt: now.Add(l.window),
		}

		return true, l.window
	}

	remaining :=
		entry.resetAt.Sub(now)

	if entry.count >= l.limit {
		return false, remaining
	}

	entry.count++

	l.entries[key] = entry

	return true, remaining
}
