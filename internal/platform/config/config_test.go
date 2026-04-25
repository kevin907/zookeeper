package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/platform/config"
)

var envKeys = []string{
	"APP_ENV", "HTTP_ADDR", "HTTP_READ_TIMEOUT", "HTTP_WRITE_TIMEOUT",
	"HTTP_SHUTDOWN_GRACE", "POSTGRES_URL", "POSTGRES_MAX_CONNS",
	"POSTGRES_MIN_CONNS", "POSTGRES_MAX_CONN_LIFETIME",
	"POSTGRES_MAX_CONN_IDLE_TIME", "SOLVER", "LOG_LEVEL",
}

// unsetAll clears every env var Config reads and restores them on cleanup,
// so the defaults test is immune to whatever shell it was launched in.
func unsetAll(t *testing.T) {
	t.Helper()
	for _, k := range envKeys {
		old, present := os.LookupEnv(k)
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
		k, old, present := k, old, present
		t.Cleanup(func() {
			if present {
				_ = os.Setenv(k, old)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	unsetAll(t)

	c, err := config.Load()
	require.NoError(t, err)

	require.Equal(t, "dev", c.AppEnv)
	require.Equal(t, ":8080", c.HTTPAddr)
	require.Equal(t, 5*time.Second, c.HTTPReadTimeout)
	require.Equal(t, 10*time.Second, c.HTTPWriteTimeout)
	require.Equal(t, 15*time.Second, c.HTTPShutdownGrace)
	require.Equal(t, "", c.PostgresURL)
	require.Equal(t, int32(10), c.PostgresMaxConns)
	require.Equal(t, int32(2), c.PostgresMinConns)
	require.Equal(t, time.Hour, c.PostgresMaxConnLifetime)
	require.Equal(t, 30*time.Minute, c.PostgresMaxConnIdleTime)
	require.Equal(t, "greedy", c.Solver)
	require.Equal(t, "info", c.LogLevel)
}

func TestLoad_Overrides(t *testing.T) {
	unsetAll(t)
	t.Setenv("APP_ENV", "prod")
	t.Setenv("HTTP_ADDR", ":9999")
	t.Setenv("POSTGRES_URL", "postgres://user:pw@host:5432/db")
	t.Setenv("POSTGRES_MAX_CONNS", "25")
	t.Setenv("SOLVER", "maxpop")

	c, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, "prod", c.AppEnv)
	require.Equal(t, ":9999", c.HTTPAddr)
	require.Equal(t, "postgres://user:pw@host:5432/db", c.PostgresURL)
	require.Equal(t, int32(25), c.PostgresMaxConns)
	require.Equal(t, "maxpop", c.Solver)
}

func TestLoad_ReturnsErrorOnInvalidDuration(t *testing.T) {
	unsetAll(t)
	t.Setenv("HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := config.Load()
	require.Error(t, err)
}
