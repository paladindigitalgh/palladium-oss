package encryption_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/encryption"
)

func validKey() []byte {
	return make([]byte, encryption.KeySize) // an all-zero key is fine for tests; never used for anything real
}

func TestNewAESGCMEncryptorRejectsWrongKeySize(t *testing.T) {
	cases := []int{0, 1, 16, 24, 31, 33, 64}

	for _, size := range cases {
		_, err := encryption.NewAESGCMEncryptor(make([]byte, size))
		if !errors.Is(err, encryption.ErrInvalidKeySize) {
			t.Errorf("NewAESGCMEncryptor(%d bytes) error = %v, want %v", size, err, encryption.ErrInvalidKeySize)
		}
	}
}

func TestNewAESGCMEncryptorAcceptsExactly32Bytes(t *testing.T) {
	if _, err := encryption.NewAESGCMEncryptor(validKey()); err != nil {
		t.Fatalf("NewAESGCMEncryptor(32 bytes) = %v, want nil", err)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	cases := []string{
		"a simple password",
		"-----BEGIN OPENSSH PRIVATE KEY-----\nMIIEow...\n-----END OPENSSH PRIVATE KEY-----",
		"unicode: пароль 密码 🔒",
		"a", // single character
	}

	for _, plaintext := range cases {
		ciphertext, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt(%q) = %v", plaintext, err)
		}
		if ciphertext == plaintext {
			t.Errorf("Encrypt(%q) returned the plaintext unchanged", plaintext)
		}

		got, err := enc.Decrypt(ciphertext)
		if err != nil {
			t.Fatalf("Decrypt() = %v", err)
		}
		if got != plaintext {
			t.Errorf("round trip = %q, want %q", got, plaintext)
		}
	}
}

// TestEncryptProducesDifferentCiphertextEachTime proves the same
// plaintext encrypted twice does not produce identical ciphertext — the
// whole point of a random nonce (see this package's doc comment,
// "Ciphertext format").
func TestEncryptProducesDifferentCiphertextEachTime(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	first, err := enc.Encrypt("same plaintext")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	second, err := enc.Encrypt("same plaintext")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	if first == second {
		t.Error("Encrypt() produced identical ciphertext for the same plaintext twice")
	}
}

func TestEncryptEmptyStringReturnsEmptyString(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	got, err := enc.Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt(\"\") = %v", err)
	}
	if got != "" {
		t.Errorf("Encrypt(\"\") = %q, want \"\"", got)
	}
}

func TestDecryptEmptyStringReturnsEmptyString(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	got, err := enc.Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt(\"\") = %v", err)
	}
	if got != "" {
		t.Errorf("Decrypt(\"\") = %q, want \"\"", got)
	}
}

func TestDecryptRejectsInvalidBase64(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	_, err = enc.Decrypt("not valid base64!!!")
	if err == nil {
		t.Fatal("Decrypt() = nil error, want an error for invalid base64")
	}
}

func TestDecryptRejectsCiphertextTooShortForNonce(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	tooShort := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = enc.Decrypt(tooShort)
	if !errors.Is(err, encryption.ErrCiphertextTooShort) {
		t.Fatalf("Decrypt() error = %v, want %v", err, encryption.ErrCiphertextTooShort)
	}
}

// TestDecryptRejectsTamperedCiphertext proves GCM's authentication
// check actually does something: flipping a byte in a genuine
// ciphertext must fail to decrypt, not silently return corrupted
// plaintext.
func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	ciphertext, err := enc.Encrypt("secret value")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		t.Fatalf("test setup: decode ciphertext: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // flip the last byte
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := enc.Decrypt(tampered); err == nil {
		t.Fatal("Decrypt() = nil error, want an error for tampered ciphertext")
	}
}

// TestDecryptRejectsWrongKey proves a ciphertext encrypted under one key
// cannot be decrypted under a different one — the entire premise of
// PALLADIUM_MASTER_KEY actually protecting stored secrets.
func TestDecryptRejectsWrongKey(t *testing.T) {
	encA, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}
	otherKey := make([]byte, encryption.KeySize)
	otherKey[0] = 1 // different from validKey()'s all-zero key
	encB, err := encryption.NewAESGCMEncryptor(otherKey)
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	ciphertext, err := encA.Encrypt("secret value")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	if _, err := encB.Decrypt(ciphertext); err == nil {
		t.Fatal("Decrypt() with the wrong key = nil error, want an error")
	}
}

func TestNewAESGCMEncryptorFromBase64KeyRejectsEmptyString(t *testing.T) {
	_, err := encryption.NewAESGCMEncryptorFromBase64Key("")
	if !errors.Is(err, encryption.ErrEmptyMasterKey) {
		t.Fatalf("error = %v, want %v", err, encryption.ErrEmptyMasterKey)
	}
}

func TestNewAESGCMEncryptorFromBase64KeyRejectsInvalidBase64(t *testing.T) {
	_, err := encryption.NewAESGCMEncryptorFromBase64Key("not valid base64!!!")
	if err == nil {
		t.Fatal("error = nil, want an error for invalid base64")
	}
}

func TestNewAESGCMEncryptorFromBase64KeyRejectsWrongDecodedSize(t *testing.T) {
	tooShort := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, err := encryption.NewAESGCMEncryptorFromBase64Key(tooShort)
	if !errors.Is(err, encryption.ErrInvalidKeySize) {
		t.Fatalf("error = %v, want %v", err, encryption.ErrInvalidKeySize)
	}
}

func TestNewAESGCMEncryptorFromBase64KeyAcceptsValidKey(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(validKey())
	enc, err := encryption.NewAESGCMEncryptorFromBase64Key(encoded)
	if err != nil {
		t.Fatalf("NewAESGCMEncryptorFromBase64Key() = %v", err)
	}

	ciphertext, err := enc.Encrypt("round trip through the base64 constructor")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	got, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() = %v", err)
	}
	if got != "round trip through the base64 constructor" {
		t.Errorf("round trip = %q, want the original plaintext", got)
	}
}

// TestCiphertextDoesNotContainPlaintext is a sanity check on the whole
// point of this package: the base64 output must not simply be the
// plaintext re-encoded — i.e. Encrypt must have actually transformed it.
func TestCiphertextDoesNotContainPlaintext(t *testing.T) {
	enc, err := encryption.NewAESGCMEncryptor(validKey())
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor() = %v", err)
	}

	plaintext := "a very recognizable secret marker value"
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}

	if strings.Contains(ciphertext, plaintext) {
		t.Error("ciphertext contains the plaintext verbatim")
	}
}
