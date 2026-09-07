// Package config loads service configuration from the environment.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	HTTPPort           int
	HTTPAllowedOrigins []string

	SQLHost         string
	SQLPort         int
	SQLUser         string
	SQLPassword     string
	SQLDatabaseName string
	SQLSchemaName   string
	SQLSSLMode      string
}

// Load reads configuration from environment variables, falling back to values that make
// `go run ./cmd/server` work against the compose Postgres with no setup.
func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("HTTP_PORT", 8081)
	// Empty by default: the console reaches this service from the SvelteKit *server*,
	// so no browser origin is involved and no CORS handling is required.
	v.SetDefault("HTTP_ALLOWED_ORIGINS", "")

	v.SetDefault("SQL_HOST", "localhost")
	v.SetDefault("SQL_PORT", 5432)
	v.SetDefault("SQL_USER", "dispatch")
	v.SetDefault("SQL_PASSWORD", "dispatch")
	v.SetDefault("SQL_DATABASE_NAME", "dispatch")
	v.SetDefault("SQL_SCHEMA_NAME", "dispatch")
	v.SetDefault("SQL_SSL_MODE", "disable")

	cfg := &Config{
		HTTPPort:        v.GetInt("HTTP_PORT"),
		SQLHost:         v.GetString("SQL_HOST"),
		SQLPort:         v.GetInt("SQL_PORT"),
		SQLUser:         v.GetString("SQL_USER"),
		SQLPassword:     v.GetString("SQL_PASSWORD"),
		SQLDatabaseName: v.GetString("SQL_DATABASE_NAME"),
		SQLSchemaName:   v.GetString("SQL_SCHEMA_NAME"),
		SQLSSLMode:      v.GetString("SQL_SSL_MODE"),
	}
	if origins := v.GetString("HTTP_ALLOWED_ORIGINS"); origins != "" {
		cfg.HTTPAllowedOrigins = strings.Split(origins, ",")
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	// The schema name is interpolated into DDL, so it must not be attacker-shaped even
	// though it comes from the operator's own environment.
	if !isIdentifier(c.SQLSchemaName) {
		return fmt.Errorf("SQL_SCHEMA_NAME %q is not a valid SQL identifier", c.SQLSchemaName)
	}
	if c.HTTPPort <= 0 || c.HTTPPort > 65535 {
		return fmt.Errorf("HTTP_PORT %d is out of range", c.HTTPPort)
	}
	return nil
}

func isIdentifier(s string) bool {
	if s == "" || len(s) > 63 {
		return false
	}
	for i, r := range s {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if isLetter || r == '_' {
			continue
		}
		if isDigit && i > 0 {
			continue
		}
		return false
	}
	return true
}

// DSN renders the PostgreSQL connection string.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.SQLHost, c.SQLPort, c.SQLUser, c.SQLPassword, c.SQLDatabaseName, c.SQLSSLMode,
	)
}
