package config

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func resetFlags(t *testing.T) {
	t.Helper()

	oldArgs := os.Args
	os.Args = []string{oldArgs[0]}
	t.Cleanup(func() { os.Args = oldArgs })

	old := pflag.CommandLine
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	t.Cleanup(func() { pflag.CommandLine = old })
}

func setEnvEmpty(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
	}
}

func setEnvMap(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	resetFlags(t)

	setEnvEmpty(t,
		"PORT",
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
		"POSTGRES_PASSWORD_FILE",
		"DELIVERY_AUTO_RELEASE_INTERVAL",
		"ORDER_SERVICE_HOST",
		"ORDER_GATEWAY_MAX_ATTEMPTS", "ORDER_GATEWAY_BASE_DELAY", "ORDER_GATEWAY_MAX_DELAY",
	)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, DefaultPort(), cfg.Port)

	require.Equal(t, DefaultDB(), cfg.DB)

	require.Equal(t, DefaultDelivery(), cfg.Delivery)
	require.Equal(t, DefaultOrderServiceHost(), cfg.OrderService)
	require.Equal(t, DefaultOrdersGateway(), cfg.OrdersGateway)
}

func TestLoad_EnvOverrides(t *testing.T) {
	resetFlags(t)

	setEnvMap(t, map[string]string{
		"PORT":                           "9090",
		"POSTGRES_HOST":                  "db",
		"POSTGRES_PORT":                  "15432",
		"POSTGRES_USER":                  "u",
		"POSTGRES_PASSWORD":              "p",
		"POSTGRES_PASSWORD_FILE":         "",
		"POSTGRES_DB":                    "service",
		"DELIVERY_AUTO_RELEASE_INTERVAL": "30s",
		"ORDER_SERVICE_HOST":             "service-order:50051",
		"ORDER_GATEWAY_MAX_ATTEMPTS":     "5",
		"ORDER_GATEWAY_BASE_DELAY":       "150ms",
		"ORDER_GATEWAY_MAX_DELAY":        "2s",
	})

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, 9090, cfg.Port)
	require.Equal(t, DB{
		Host: "db", Port: "15432", User: "u", Pass: "p", Name: "service",
	}, cfg.DB)
	require.Equal(t, Delivery{
		AutoReleaseInterval: 30 * time.Second,
	}, cfg.Delivery)
	require.Equal(t, "service-order:50051", cfg.OrderService)
	require.Equal(t, OrdersGateway{
		MaxAttempts: 5,
		BaseDelay:   150 * time.Millisecond,
		MaxDelay:    2 * time.Second,
	}, cfg.OrdersGateway)
}

func TestLoad_InvalidPort(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "70000")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "10s")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidPostgresPort(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "8080")
	t.Setenv("POSTGRES_PORT", "not-a-number")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "10s")
	t.Setenv("POSTGRES_PASSWORD_FILE", "")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidReleaseInterval(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "8080")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "bad-interval")
	t.Setenv("POSTGRES_PASSWORD_FILE", "")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidOrderGatewayMaxAttempts(t *testing.T) {
	resetFlags(t)
	setEnvEmpty(t,
		"PORT",
		"POSTGRES_PASSWORD_FILE",
		"DELIVERY_AUTO_RELEASE_INTERVAL",
		"ORDER_GATEWAY_MAX_ATTEMPTS", "ORDER_GATEWAY_BASE_DELAY", "ORDER_GATEWAY_MAX_DELAY",
	)
	t.Setenv("ORDER_GATEWAY_MAX_ATTEMPTS", "0")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidOrderGatewayBaseDelay(t *testing.T) {
	resetFlags(t)
	setEnvEmpty(t,
		"PORT",
		"POSTGRES_PASSWORD_FILE",
		"DELIVERY_AUTO_RELEASE_INTERVAL",
		"ORDER_GATEWAY_MAX_ATTEMPTS", "ORDER_GATEWAY_BASE_DELAY", "ORDER_GATEWAY_MAX_DELAY",
	)
	t.Setenv("ORDER_GATEWAY_BASE_DELAY", "bad")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidOrderGatewayMaxDelayLessThanBase(t *testing.T) {
	resetFlags(t)
	setEnvEmpty(t,
		"PORT",
		"POSTGRES_PASSWORD_FILE",
		"DELIVERY_AUTO_RELEASE_INTERVAL",
		"ORDER_GATEWAY_MAX_ATTEMPTS", "ORDER_GATEWAY_BASE_DELAY", "ORDER_GATEWAY_MAX_DELAY",
	)
	t.Setenv("ORDER_GATEWAY_BASE_DELAY", "200ms")
	t.Setenv("ORDER_GATEWAY_MAX_DELAY", "100ms")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_FlagsParseError(t *testing.T) {
	oldArgs := os.Args
	oldCommandLine := pflag.CommandLine

	defer func() {
		os.Args = oldArgs
		pflag.CommandLine = oldCommandLine
	}()

	t.Setenv("PORT", "")

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pflag.CommandLine = fs
	os.Args = []string{"cmd", "--port=not-a-number"}

	cfg, err := Load()

	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "parse flags")
}

func TestLoad_PostgresPasswordFile_ReadError_ReturnsError(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "")

	secretDir := t.TempDir() + "/secret-dir"
	require.NoError(t, os.Mkdir(secretDir, 0o755))

	setEnvMap(t, map[string]string{
		"POSTGRES_PASSWORD":      "from-env",
		"POSTGRES_PASSWORD_FILE": secretDir,
	})

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "read POSTGRES_PASSWORD_FILE")
}

func TestLoad_InvalidKafkaBrokers_EmptyAfterTrim(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "")
	t.Setenv("POSTGRES_PASSWORD_FILE", "")

	t.Setenv("KAFKA_BROKERS", " , ,   , ")

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "invalid KAFKA_BROKERS")
}

func TestParseRateLimit_InvalidEnabled(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "notabool")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_ENABLED")
}

func TestParseRateLimit_Disabled_ReturnsDefaultWithEnabledFalse(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "false")

	got, err := parseRateLimit()
	require.NoError(t, err)

	want := defaultRateLimit
	want.Enabled = false
	require.Equal(t, want, got)
}

func TestParseRateLimit_InvalidRate_Format(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "oops")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_RATE")
}

func TestParseRateLimit_InvalidRate_Validate(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "0")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_RATE")
}

func TestParseRateLimit_InvalidBurst_Format(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "5")
	t.Setenv("RATE_LIMIT_BURST", "oops")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_BURST")
}

func TestParseRateLimit_InvalidTTL_Format(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "5")
	t.Setenv("RATE_LIMIT_BURST", "5")
	t.Setenv("RATE_LIMIT_TTL", "oops")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_TTL")
}

func TestParseRateLimit_InvalidTTL_ValidateNegative(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "5")
	t.Setenv("RATE_LIMIT_BURST", "5")
	t.Setenv("RATE_LIMIT_TTL", "-1s")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_TTL")
}

func TestParseRateLimit_InvalidMaxBuckets_Format(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "5")
	t.Setenv("RATE_LIMIT_BURST", "5")
	t.Setenv("RATE_LIMIT_TTL", "10s")
	t.Setenv("RATE_LIMIT_MAX_BUCKETS", "oops")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_MAX_BUCKETS")
}

func TestParseRateLimit_InvalidMaxBuckets_ValidateNegative(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("RATE_LIMIT_RATE", "5")
	t.Setenv("RATE_LIMIT_BURST", "5")
	t.Setenv("RATE_LIMIT_TTL", "10s")
	t.Setenv("RATE_LIMIT_MAX_BUCKETS", "-1")

	_, err := parseRateLimit()
	require.Error(t, err)
	require.Contains(t, err.Error(), "RATE_LIMIT_MAX_BUCKETS")
}
