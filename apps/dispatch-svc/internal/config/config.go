// Package config loads service configuration from the environment.
package config

import (
	"fmt"
	"strings"
	"time"

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

	// IAMS / Keycloak. The issuer URL these compose into is what
	// aohhttp.BearerAuth validates every bearer token against.
	IAMSKeycloakHost  string
	IAMSKeycloakPort  int
	IAMSKeycloakRealm string

	// GISURL is gis-service's base URL — the only downstream this service has for
	// entity data. It carries no RTUS endpoint, topic or credential: gis-service
	// publishes entity changes to the `gis` RTUS map itself.
	GISURL string

	// How often a projection is retried. The real bound on a delivery is the remaining
	// lifetime of the operator token it carries (design.md D2a) — ProjectionAttempts is
	// a safety cap, and 0 (the default) means "as many as fit before that token
	// expires". Capping by attempt count instead would strand a row during a
	// gis-service outage of a few tens of seconds, while the token was still good.
	//
	// There is deliberately no poll interval: the projection is triggered by the write
	// that produced it, so a timer that woke with no credential could deliver nothing.
	ProjectionAttempts   int
	ProjectionBackoff    time.Duration
	ProjectionMaxBackoff time.Duration
}

// IssuerURL is the realm the bearer token must come from, e.g.
// http://iams-keycloak.127.0.0.1.nip.io/realms/aoh.
func (c *Config) IssuerURL() string {
	return fmt.Sprintf("%s:%d/realms/%s",
		strings.TrimSuffix(c.IAMSKeycloakHost, "/"), c.IAMSKeycloakPort, c.IAMSKeycloakRealm)
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

	// The scaffold's own key names. IAMS_KEYCLOAK_HOST carries the scheme: without it
	// the composed issuer URL is unparseable and every token validation fails.
	v.SetDefault("IAMS_KEYCLOAK_HOST", "http://iams-keycloak")
	v.SetDefault("IAMS_KEYCLOAK_PORT", 8080)
	v.SetDefault("IAMS_KEYCLOAK_REALM", "aoh")

	v.SetDefault("GIS_URL", "http://gis-service:8080")

	// 0 = bounded only by the operator token's remaining lifetime.
	v.SetDefault("PROJECTION_ATTEMPTS", 0)
	v.SetDefault("PROJECTION_BACKOFF", "500ms")
	v.SetDefault("PROJECTION_MAX_BACKOFF", "15s")

	cfg := &Config{
		HTTPPort:        v.GetInt("HTTP_PORT"),
		SQLHost:         v.GetString("SQL_HOST"),
		SQLPort:         v.GetInt("SQL_PORT"),
		SQLUser:         v.GetString("SQL_USER"),
		SQLPassword:     v.GetString("SQL_PASSWORD"),
		SQLDatabaseName: v.GetString("SQL_DATABASE_NAME"),
		SQLSchemaName:   v.GetString("SQL_SCHEMA_NAME"),
		SQLSSLMode:      v.GetString("SQL_SSL_MODE"),

		IAMSKeycloakHost:  v.GetString("IAMS_KEYCLOAK_HOST"),
		IAMSKeycloakPort:  v.GetInt("IAMS_KEYCLOAK_PORT"),
		IAMSKeycloakRealm: v.GetString("IAMS_KEYCLOAK_REALM"),

		GISURL: v.GetString("GIS_URL"),

		ProjectionAttempts:   v.GetInt("PROJECTION_ATTEMPTS"),
		ProjectionBackoff:    v.GetDuration("PROJECTION_BACKOFF"),
		ProjectionMaxBackoff: v.GetDuration("PROJECTION_MAX_BACKOFF"),
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
	// A host with no scheme composes into an issuer URL that cannot be fetched, and the
	// failure surfaces later as an opaque 401 on every request rather than as a
	// configuration error here.
	if !strings.HasPrefix(c.IAMSKeycloakHost, "http://") && !strings.HasPrefix(c.IAMSKeycloakHost, "https://") {
		return fmt.Errorf("IAMS_KEYCLOAK_HOST %q must include the scheme", c.IAMSKeycloakHost)
	}
	if c.IAMSKeycloakRealm == "" {
		return fmt.Errorf("IAMS_KEYCLOAK_REALM must not be empty")
	}
	if c.GISURL == "" {
		return fmt.Errorf("GIS_URL must not be empty")
	}
	if c.ProjectionAttempts < 0 {
		return fmt.Errorf("PROJECTION_ATTEMPTS %d must not be negative (0 = token-bounded)", c.ProjectionAttempts)
	}
	if c.ProjectionBackoff <= 0 {
		return fmt.Errorf("PROJECTION_BACKOFF must be positive")
	}
	if c.ProjectionMaxBackoff < c.ProjectionBackoff {
		return fmt.Errorf("PROJECTION_MAX_BACKOFF must be at least PROJECTION_BACKOFF")
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
