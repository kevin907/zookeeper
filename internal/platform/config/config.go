// Package config loads environment-driven configuration once at startup.
package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds every environment variable the application reads.
// Loaded via envconfig so defaults and parsing are struct-tag driven.
type Config struct {
	AppEnv string `envconfig:"APP_ENV" default:"dev"`

	HTTPAddr          string        `envconfig:"HTTP_ADDR" default:":8080"`
	HTTPReadTimeout   time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"5s"`
	HTTPWriteTimeout  time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"10s"`
	HTTPShutdownGrace time.Duration `envconfig:"HTTP_SHUTDOWN_GRACE" default:"15s"`

	PostgresURL             string        `envconfig:"POSTGRES_URL"`
	PostgresMaxConns        int32         `envconfig:"POSTGRES_MAX_CONNS" default:"10"`
	PostgresMinConns        int32         `envconfig:"POSTGRES_MIN_CONNS" default:"2"`
	PostgresMaxConnLifetime time.Duration `envconfig:"POSTGRES_MAX_CONN_LIFETIME" default:"1h"`
	PostgresMaxConnIdleTime time.Duration `envconfig:"POSTGRES_MAX_CONN_IDLE_TIME" default:"30m"`

	Solver   string `envconfig:"SOLVER" default:"greedy"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`
}

// Load reads the process environment into a Config, applying struct-tag
// defaults and surfacing any parse error.
func Load() (Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	return c, nil
}
