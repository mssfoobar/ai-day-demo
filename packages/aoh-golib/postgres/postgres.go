package aohpostgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Schema   string
	SSLMode  string
}

// NewConnString construct postgres connection string
func NewConnString(c Config) string {
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}

	// Use url.URL struct to properly handle special characters in password and other fields
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%s", c.Host, c.Port),
		Path:   "/" + c.Database,
	}

	// Add query parameters
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	q.Set("search_path", c.Schema)
	u.RawQuery = q.Encode()

	return u.String()
}

// NewPostgresConn return db connection
func NewPostgresConn(connString string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// NewDbConnection return standard sql.DB connection
func NewDbConnection(connString string, driverName string) (*sql.DB, error) {
	db, err := sql.Open(driverName, connString)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// RunDbMigrate migrate sql up scripts from `migrationUrl` to db
func RunDbMigrate(migrationUrl string, connString string) error {
	m, err := migrate.New(migrationUrl, connString)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// StringToUUID convert string into *uuid.UUID. return nil when string is empty
func StringToUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

// UuidToString convert *uuid.UUID into string. return empty string when uuid is nil
func UuidToString(id *uuid.UUID) string {
	if id == nil || *id == uuid.Nil {
		return ""
	}
	return id.String()
}

// StringToTimestamptz convert string into pgtype.Timestamptz. return invalid pgtype.Timestamptz when string cannot be parsed
func StringToTimestamptz(s string) pgtype.Timestamptz {
	ts := pgtype.Timestamptz{Valid: true}
	if err := ts.Scan(s); err != nil {
		ts.Valid = false
	}
	return ts
}

// TimeToString convert time to string in RFC3339Nano format. return emtpy string if time is zero
func TimeToString(t time.Time) string {
	if !t.IsZero() {
		return t.Format(time.RFC3339Nano)
	}
	return ""
}
