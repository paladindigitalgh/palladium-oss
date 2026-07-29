// Package authentication models Palladium's Authentication domain (v1):
// a named, reusable authentication method — a username paired with
// either a password or an SSH private key — that a future
// ConnectionProfile (see internal/connectionprofile) references to
// describe how to log in to a device. This package holds only the
// domain model, field validation, and the repository interface — no
// SQL, no migrations, no HTTP CRUD — mirroring internal/catalog's own
// package exactly.
//
// This is infrastructure, not a feature — the same distinction this
// milestone's own instructions draw explicitly ("This is NOT
// diagnostics. This is NOT SSH execution. This is NOT OLT
// connectivity."). Nothing in this package opens a connection, runs a
// command, or knows what a Kontron OLT or any other device expects; it
// only records that a named credential exists.
//
// # Plaintext in memory, ciphertext at rest
//
// Password and PrivateKey hold plaintext in this Go struct — the
// "real," usable value, exactly as every other field in this codebase
// holds its real value rather than some storage-layer encoding of it.
// Encryption is a storage-representation concern, not a domain concern:
// internal/authentication/postgres encrypts Password and PrivateKey
// (via internal/platform/encryption) immediately before writing them and
// decrypts them immediately after reading them back, the same way it
// converts AuthenticationType to and from a plain TEXT column — nothing
// in this package, or in internal/authentication/service one layer up,
// ever touches ciphertext. See internal/authentication/postgres's own
// doc comment for the repository-level detail, and
// internal/authentication/httpapi's for why an HTTP response never
// echoes either field back despite the domain type holding real
// plaintext internally.
package authentication

import (
	"time"

	"github.com/google/uuid"
)

// Authentication is a single named authentication method: a username
// paired with either a password or an SSH private key (see
// AuthenticationType), reusable across any future ConnectionProfile that
// needs to log in the same way.
type Authentication struct {
	ID                 uuid.UUID
	Name               string
	AuthenticationType AuthenticationType
	Username           string
	Password           string
	PrivateKey         string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
