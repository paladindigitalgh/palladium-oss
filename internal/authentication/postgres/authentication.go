// Package postgres implements the Authentication domain's
// AuthenticationRepository against PostgreSQL using pgx directly — no
// ORM — following the exact pattern established by
// internal/catalog/postgres.CatalogRepository, with one addition: every
// method that writes or reads Password or PrivateKey passes them
// through an injected internal/platform/encryption.Encryptor first.
//
// This is the one place in the Authentication domain that ciphertext
// exists at all. Create and Update encrypt Password and PrivateKey
// immediately before the INSERT/UPDATE statement runs; Get, List, and
// the RETURNING clause of Create/Update decrypt them immediately after
// scanning a row. internal/authentication (the domain package) and
// internal/authentication/service both only ever see plaintext — see
// internal/authentication/model.go's own doc comment, "Plaintext in
// memory, ciphertext at rest," for the full reasoning.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/paladindigitalgh/palladium-oss/internal/authentication"
	"github.com/paladindigitalgh/palladium-oss/internal/database"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/clock"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/encryption"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/id"
)

// AuthenticationRepository implements
// authentication.AuthenticationRepository against PostgreSQL. See
// internal/inventory/postgres/site.go for the reasoning behind depending
// on database.Querier and injecting clock/ids, which is not repeated
// here; encryptor is this domain's own addition, injected the same way
// for the same reason — a concrete dependency a test can substitute a
// fake for (see this package's own tests).
type AuthenticationRepository struct {
	db        database.Querier
	clock     clock.Clock
	ids       id.Generator
	encryptor encryption.Encryptor
}

var _ authentication.AuthenticationRepository = (*AuthenticationRepository)(nil)

// NewAuthenticationRepository builds an AuthenticationRepository.
func NewAuthenticationRepository(db database.Querier, clock clock.Clock, ids id.Generator, encryptor encryption.Encryptor) *AuthenticationRepository {
	return &AuthenticationRepository{db: db, clock: clock, ids: ids, encryptor: encryptor}
}

// Get retrieves an Authentication by ID, or an apperror.KindNotFound
// error if none exists. The returned Authentication's Password and
// PrivateKey are plaintext — see this package's own doc comment.
func (r *AuthenticationRepository) Get(ctx context.Context, authID uuid.UUID) (authentication.Authentication, error) {
	const query = `
		SELECT id, name, authentication_type, username, password, private_key, created_at, updated_at
		FROM authentication_methods
		WHERE id = $1
	`

	a, err := r.scan(r.db.QueryRow(ctx, query, authID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authentication.Authentication{}, authenticationNotFound(authID)
		}
		return authentication.Authentication{}, translateError("get authentication", err)
	}
	return a, nil
}

// List returns every Authentication, ordered by name for stable,
// human-useful output (see the UNIQUE constraint's own index, usable for
// this ordering, in the migration).
func (r *AuthenticationRepository) List(ctx context.Context) ([]authentication.Authentication, error) {
	const query = `
		SELECT id, name, authentication_type, username, password, private_key, created_at, updated_at
		FROM authentication_methods
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, translateError("list authentications", err)
	}
	defer rows.Close()

	auths := []authentication.Authentication{}
	for rows.Next() {
		a, err := r.scan(rows)
		if err != nil {
			return nil, translateError("scan authentication row", err)
		}
		auths = append(auths, a)
	}
	if err := rows.Err(); err != nil {
		return nil, translateError("list authentications", err)
	}

	return auths, nil
}

// Create encrypts a.Password and a.PrivateKey, inserts the result, and
// returns the persisted record with Password/PrivateKey decrypted back
// to plaintext.
//
// As with CatalogRepository.Create, the repository assigns ID, CreatedAt,
// and UpdatedAt itself — any values already set on the input
// Authentication for those fields are ignored. A Name that collides with
// an existing Authentication fails with an apperror.KindConflict error
// (see translateError; authentication_methods.name is UNIQUE).
func (r *AuthenticationRepository) Create(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	encryptedPassword, encryptedPrivateKey, err := r.encryptSecrets(a)
	if err != nil {
		return authentication.Authentication{}, err
	}

	const query = `
		INSERT INTO authentication_methods (id, name, authentication_type, username, password, private_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, name, authentication_type, username, password, private_key, created_at, updated_at
	`

	now := r.clock.Now()
	created, err := r.scan(r.db.QueryRow(ctx, query,
		r.ids.New(), a.Name, string(a.AuthenticationType), a.Username, encryptedPassword, encryptedPrivateKey, now))
	if err != nil {
		return authentication.Authentication{}, translateError("create authentication", err)
	}
	return created, nil
}

// Update overwrites the mutable fields of the Authentication identified
// by a.ID — encrypting a.Password and a.PrivateKey the same way
// Create does — and returns the persisted record, or an
// apperror.KindNotFound error if it does not exist.
//
// CreatedAt cannot be altered through this method: the UPDATE statement
// below never assigns that column, and the RETURNING clause reports its
// true stored value regardless of what the input Authentication
// contained. Like every other domain's Update in this codebase, this is
// a full replace, not a partial patch: a caller that omits Password (or
// PrivateKey) is asking to overwrite the stored value with an empty
// string, exactly as omitting Description elsewhere overwrites it with
// "" — there is no "leave the existing secret unchanged" convenience
// this milestone asks for.
func (r *AuthenticationRepository) Update(ctx context.Context, a authentication.Authentication) (authentication.Authentication, error) {
	encryptedPassword, encryptedPrivateKey, err := r.encryptSecrets(a)
	if err != nil {
		return authentication.Authentication{}, err
	}

	const query = `
		UPDATE authentication_methods
		SET name = $1, authentication_type = $2, username = $3, password = $4, private_key = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, name, authentication_type, username, password, private_key, created_at, updated_at
	`

	updated, err := r.scan(r.db.QueryRow(ctx, query,
		a.Name, string(a.AuthenticationType), a.Username, encryptedPassword, encryptedPrivateKey, r.clock.Now(), a.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return authentication.Authentication{}, authenticationNotFound(a.ID)
		}
		return authentication.Authentication{}, translateError("update authentication", err)
	}
	return updated, nil
}

// Delete removes the Authentication identified by id, or returns an
// apperror.KindNotFound error if it does not exist. If a ConnectionProfile
// still references this Authentication, the delete fails with
// apperror.KindConflict instead (see errors.go's translateError and the
// ON DELETE RESTRICT foreign key in
// database/migrations/00024_connectionprofile_connection_profiles.sql).
func (r *AuthenticationRepository) Delete(ctx context.Context, authID uuid.UUID) error {
	const query = `DELETE FROM authentication_methods WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, authID)
	if err != nil {
		return translateError("delete authentication", err)
	}
	if tag.RowsAffected() == 0 {
		return authenticationNotFound(authID)
	}
	return nil
}

// encryptSecrets encrypts plaintext.Password and plaintext.PrivateKey,
// wrapping any encryption failure as apperror.KindInternal — an
// encryption failure here means PALLADIUM_MASTER_KEY is misconfigured or
// crypto/rand is unavailable, not that the caller supplied bad input
// (Authentication.Validate has already run by the time a repository
// method is called — see internal/authentication/service — so this is
// always an unexpected, server-side condition).
func (r *AuthenticationRepository) encryptSecrets(plaintext authentication.Authentication) (password, privateKey string, err error) {
	password, err = r.encryptor.Encrypt(plaintext.Password)
	if err != nil {
		return "", "", apperror.Internal("encrypt password", err)
	}
	privateKey, err = r.encryptor.Encrypt(plaintext.PrivateKey)
	if err != nil {
		return "", "", apperror.Internal("encrypt private key", err)
	}
	return password, privateKey, nil
}

func authenticationNotFound(id uuid.UUID) error {
	return apperror.NotFound(fmt.Sprintf("authentication %s not found", id))
}

// rowScanner is satisfied by both pgx.Row (QueryRow, a single row) and
// pgx.Rows (Query, iterated one row at a time via Next then Scan), so
// scan backs Get/Create/Update and List alike.
type rowScanner interface {
	Scan(dest ...any) error
}

// scan reads one row into an authentication.Authentication and decrypts
// its Password and PrivateKey back to plaintext, wrapping any
// decryption failure as apperror.KindInternal — a stored ciphertext that
// fails to decrypt means PALLADIUM_MASTER_KEY has changed since it was
// written, or the row was corrupted; either way it is not something the
// caller who asked to Get or List this record did wrong.
func (r *AuthenticationRepository) scan(row rowScanner) (authentication.Authentication, error) {
	var (
		a                                      authentication.Authentication
		authType                               string
		encryptedPassword, encryptedPrivateKey string
	)
	if err := row.Scan(&a.ID, &a.Name, &authType, &a.Username, &encryptedPassword, &encryptedPrivateKey, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return authentication.Authentication{}, err
	}
	a.AuthenticationType = authentication.AuthenticationType(authType)

	password, err := r.encryptor.Decrypt(encryptedPassword)
	if err != nil {
		return authentication.Authentication{}, apperror.Internal("decrypt password", err)
	}
	privateKey, err := r.encryptor.Decrypt(encryptedPrivateKey)
	if err != nil {
		return authentication.Authentication{}, apperror.Internal("decrypt private key", err)
	}
	a.Password = password
	a.PrivateKey = privateKey

	return a, nil
}
