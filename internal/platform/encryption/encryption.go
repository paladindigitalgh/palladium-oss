// Package encryption is Palladium's reusable secret-encryption platform
// package: a thin wrapper around AES-256-GCM for encrypting a string
// secret before it is persisted, and decrypting it back for legitimate
// use. It is infrastructure, not a feature — it has no knowledge of
// Authentication, ConnectionProfile, or any other domain that will
// eventually store a secret through it (per this milestone's explicit
// instruction: "The encryption implementation should be reusable by
// future infrastructure packages").
//
// This package has zero dependency on any business domain — only the
// standard library. It does not import internal/platform/apperror, for
// the same reason internal/platform/ssh doesn't (see that package's own
// doc comment): this package models a cryptographic primitive, not an
// HTTP-facing API boundary, so it has no reason to carry apperror's Kind
// taxonomy. A caller that does sit closer to that boundary (e.g.
// internal/authentication/postgres) is responsible for deciding how an
// encryption failure should ultimately be reported.
//
// # Algorithm and key
//
// Encryption is AES-256-GCM: AES-256 for the cipher, GCM for authenticated
// encryption (so a tampered or corrupted ciphertext fails to decrypt
// loudly, rather than silently producing garbage plaintext). The key is
// always 32 bytes (AES-256's key size) — see NewAESGCMEncryptorFromBase64Key
// for how the PALLADIUM_MASTER_KEY environment variable, a base64 string,
// becomes those 32 bytes, and NewAESGCMEncryptor if a caller already has
// raw key bytes.
//
// # Ciphertext format
//
// Encrypt returns a base64-encoded string — a random 12-byte GCM nonce
// followed by GCM's sealed output (ciphertext + 16-byte authentication
// tag) — so the result is safe to store in a plain TEXT column, matching
// this codebase's established preference for TEXT over BYTEA wherever a
// column's content is naturally string-shaped (see e.g. every
// description column in this codebase's migrations). The nonce does not
// need to be kept secret; it exists so encrypting the same plaintext
// twice never produces the same ciphertext twice, which is what makes
// GCM secure to reuse a single key across many separate secrets, as
// every future caller of this package will.
//
// # Empty string is not encrypted
//
// Both Encrypt("") and Decrypt("") return "" with no error, rather than
// producing (or requiring) a real ciphertext. This is a deliberate
// convenience for optional secret fields — e.g.
// authentication.Authentication.PrivateKey is empty for a Password-type
// Authentication record — so a caller never needs to special-case "this
// field legitimately has no value" before calling either method, and a
// stored empty column round-trips as empty rather than as the
// (meaningless) encryption of an empty string.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// KeySize is the required length, in bytes, of the key passed to
// NewAESGCMEncryptor — 32 bytes, AES-256's key size.
const KeySize = 32

// Sentinel errors this package's own logic returns. Anything else
// (a malformed base64 master key, a corrupted or tampered ciphertext) is
// returned wrapped via fmt.Errorf's %w rather than translated into a
// sentinel of this package's own — see this package's doc comment for
// why that wrapping stops short of apperror translation.
var (
	// ErrInvalidKeySize means a key that was not exactly KeySize (32)
	// bytes was passed to NewAESGCMEncryptor.
	ErrInvalidKeySize = errors.New("encryption: key must be exactly 32 bytes (AES-256)")

	// ErrEmptyMasterKey means NewAESGCMEncryptorFromBase64Key was called
	// with an empty string — there is no key to decode at all.
	ErrEmptyMasterKey = errors.New("encryption: master key must not be empty")

	// ErrCiphertextTooShort means Decrypt was given a string that,
	// once base64-decoded, is shorter than a GCM nonce — it cannot
	// possibly be a value Encrypt produced.
	ErrCiphertextTooShort = errors.New("encryption: ciphertext is too short to contain a nonce")
)

// Encryptor encrypts and decrypts string secrets. It is an interface —
// with aesGCMEncryptor as its one implementation in this package — for
// the same reason every injected dependency in this codebase is (see
// e.g. internal/platform/clock.Clock): it lets a future package's
// service- or repository-layer tests supply a fake Encryptor instead of
// exercising real AES-GCM, and it keeps the door open for a different
// backing (e.g. an actual KMS/Vault-backed implementation — see this
// milestone's explicit "OUT OF SCOPE: Vault integration," which this
// interface deliberately does not foreclose on for a future milestone)
// without any caller changing.
type Encryptor interface {
	// Encrypt returns plaintext's ciphertext, base64-encoded. See this
	// package's doc comment, "Empty string is not encrypted," for
	// Encrypt("")'s behavior.
	Encrypt(plaintext string) (string, error)

	// Decrypt reverses Encrypt. It returns an error if ciphertext is not
	// valid base64, is too short to contain a nonce, or fails GCM's
	// authentication check (wrong key, corrupted data, or truncated
	// data). See this package's doc comment, "Empty string is not
	// encrypted," for Decrypt("")'s behavior.
	Decrypt(ciphertext string) (string, error)
}

// aesGCMEncryptor is Encryptor's one implementation, backed by
// crypto/aes and crypto/cipher's GCM mode.
type aesGCMEncryptor struct {
	gcm cipher.AEAD
}

var _ Encryptor = (*aesGCMEncryptor)(nil)

// NewAESGCMEncryptor builds an Encryptor from a raw 32-byte AES-256 key.
// Most callers should use NewAESGCMEncryptorFromBase64Key instead, which
// decodes PALLADIUM_MASTER_KEY's base64 representation into these 32
// bytes; this constructor exists for callers (and tests) that already
// have raw key bytes.
func NewAESGCMEncryptor(key []byte) (Encryptor, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encryption: create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encryption: create GCM mode: %w", err)
	}

	return &aesGCMEncryptor{gcm: gcm}, nil
}

// NewAESGCMEncryptorFromBase64Key decodes encoded (a base64 string — the
// form PALLADIUM_MASTER_KEY is set to; see internal/config) and builds
// an Encryptor from the result. It returns ErrEmptyMasterKey if encoded
// is empty, a wrapped error if it is not valid base64, and
// ErrInvalidKeySize (via NewAESGCMEncryptor) if the decoded key is not
// exactly 32 bytes.
func NewAESGCMEncryptorFromBase64Key(encoded string) (Encryptor, error) {
	if encoded == "" {
		return nil, ErrEmptyMasterKey
	}

	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("encryption: decode master key: %w", err)
	}

	return NewAESGCMEncryptor(key)
}

// Encrypt implements Encryptor.
func (e *aesGCMEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encryption: generate nonce: %w", err)
	}

	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt implements Encryptor.
func (e *aesGCMEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("encryption: decode ciphertext: %w", err)
	}

	nonceSize := e.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("encryption: decrypt: %w", err)
	}

	return string(plaintext), nil
}
