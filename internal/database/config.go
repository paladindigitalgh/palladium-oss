// Package database is the PostgreSQL persistence foundation: connection
// lifecycle management, the interface boundary future repositories will
// depend on, and a health check adapter. It defines no repositories and no
// domain schema of its own.
package database

import (
	"fmt"
	"net/url"
	"time"
)

// Config holds everything needed to open a connection pool. It is
// deliberately decoupled from internal/config so this package can be
// imported and tested independently; the composition root (cmd/*) maps
// config.DatabaseConfig onto this type.
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

// DSN renders cfg as a postgres:// connection string, percent-encoding the
// credentials via net/url so passwords containing reserved characters are
// never mishandled.
func (c Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   "/" + c.Database,
	}

	q := url.Values{}
	if c.SSLMode != "" {
		q.Set("sslmode", c.SSLMode)
	}
	u.RawQuery = q.Encode()

	return u.String()
}
