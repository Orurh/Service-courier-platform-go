package config_test

import (
	"io"
	"os"
	"testing"
	"time"

	"course-go-avito-Orurh/internal/config"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func resetFlags(t *testing.T) {
	t.Helper()
	old := pflag.CommandLine
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	t.Cleanup(func() {
		pflag.CommandLine = old
	})
}

func TestLoad_Defaults(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "")
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_PASSWORD", "")
	t.Setenv("POSTGRES_DB", "")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, 8080, cfg.Port)

	require.Equal(t, "127.0.0.1", cfg.DB.Host)
	require.Equal(t, "5432", cfg.DB.Port)
	require.Equal(t, "myuser", cfg.DB.User)
	require.Equal(t, "mypassword", cfg.DB.Pass)
	require.Equal(t, "test_db", cfg.DB.Name)

	require.Equal(t, 10*time.Second, cfg.Delivery.AutoReleaseInterval)
}

func TestLoad_EnvOverrides(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "9090")
	t.Setenv("POSTGRES_HOST", "db")
	t.Setenv("POSTGRES_PORT", "15432")
	t.Setenv("POSTGRES_USER", "u")
	t.Setenv("POSTGRES_PASSWORD", "p")
	t.Setenv("POSTGRES_DB", "service")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "30s")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, 9090, cfg.Port)
	require.Equal(t, "db", cfg.DB.Host)
	require.Equal(t, "15432", cfg.DB.Port)
	require.Equal(t, "u", cfg.DB.User)
	require.Equal(t, "p", cfg.DB.Pass)
	require.Equal(t, "service", cfg.DB.Name)
	require.Equal(t, 30*time.Second, cfg.Delivery.AutoReleaseInterval)
}

func TestLoad_InvalidPort(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "70000")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "10s")

	cfg, err := config.Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidPostgresPort(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "8080")
	t.Setenv("POSTGRES_PORT", "not-a-number")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "10s")

	cfg, err := config.Load()
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidReleaseInterval(t *testing.T) {
	resetFlags(t)

	t.Setenv("PORT", "8080")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("DELIVERY_AUTO_RELEASE_INTERVAL", "bad-interval")

	cfg, err := config.Load()
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

	cfg, err := config.Load()

	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "parse flags")
}
