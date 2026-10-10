package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr                   string
	HTTPReadHeaderTimeout      time.Duration
	HTTPIdleTimeout            time.Duration
	ShutdownTimeout            time.Duration
	RTSPAllowPrivateNetworks   bool
	RTSPAllowedHosts           []string
	MaxActiveStreams           int
	MaxViewersPerStream        int
	ExpensiveRequestsPerMinute int
	TrustProxyHeaders          bool
}

func Load() Config {
	return Config{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		HTTPReadHeaderTimeout: 5 * time.Second,
		HTTPIdleTimeout:       60 * time.Second,
		ShutdownTimeout:       10 * time.Second,

		RTSPAllowPrivateNetworks: getEnvBool(
			"RTSP_ALLOW_PRIVATE_NETWORKS",
			false,
		),

		RTSPAllowedHosts: getEnvList(
			"RTSP_ALLOWED_HOSTS",
		),
		MaxActiveStreams: getEnvPositiveInt(
			"MAX_ACTIVE_STREAMS",
			4,
		),
		MaxViewersPerStream: getEnvPositiveInt(
			"MAX_VIEWERS_PER_STREAM",
			8,
		),
		ExpensiveRequestsPerMinute: getEnvPositiveInt(
			"EXPENSIVE_REQUESTS_PER_MINUTE",
			10,
		),

		TrustProxyHeaders: getEnvBool(
			"TRUST_PROXY_HEADERS",
			false,
		),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvBool(
	key string,
	fallback bool,
) bool {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	parsed, err :=
		strconv.ParseBool(value)

	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvList(
	key string,
) []string {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return nil
	}

	parts := strings.Split(
		value,
		",",
	)

	result := make(
		[]string,
		0,
		len(parts),
	)

	for _, part := range parts {
		host := strings.ToLower(
			strings.TrimSpace(part),
		)

		if host != "" {
			result = append(
				result,
				host,
			)
		}
	}

	return result
}

func getEnvPositiveInt(
	key string,
	fallback int,
) int {
	value := strings.TrimSpace(
		os.Getenv(key),
	)

	if value == "" {
		return fallback
	}

	parsed, err :=
		strconv.Atoi(value)

	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
