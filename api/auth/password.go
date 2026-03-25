package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Password policy uses Argon2id (OWASP-preferred memory-hard function).
// Tunables are centralized here; bump memory/time in a future migration note if ops require it.
// Rationale: argon2id over bcrypt for GPU-resistant hashing without a third-party module beyond x/crypto (already in module graph).
const (
	argon2Time      = 3
	argon2MemoryKiB = 64 * 1024 // 64 MiB
	argon2Threads   = 2
	argon2KeyLen    = 32
	argon2SaltLen   = 16
)

const (
	passwordMinLen = 8
	passwordMaxLen = 200
)

var (
	ErrPasswordTooShort = errors.New("password too short")
	ErrPasswordTooLong  = errors.New("password too long")
	ErrInvalidHash      = errors.New("invalid password hash encoding")
)

// HashPassword returns an encoded Argon2id hash string (not the plaintext password).
func HashPassword(password string) (string, error) {
	if len(password) < passwordMinLen {
		return "", ErrPasswordTooShort
	}
	if len(password) > passwordMaxLen {
		return "", ErrPasswordTooLong
	}
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2MemoryKiB, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2MemoryKiB, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword checks plaintext against an encoded Argon2id hash from HashPassword.
func VerifyPassword(password, encoded string) (bool, error) {
	if encoded == "" {
		return false, nil
	}
	salt, hash, err := parseArgon2IDEncoded(encoded)
	if err != nil {
		return false, err
	}
	if len(password) > passwordMaxLen {
		return false, nil
	}
	computed := argon2.IDKey([]byte(password), salt, argon2Time, argon2MemoryKiB, argon2Threads, uint32(len(hash)))
	if subtle.ConstantTimeCompare(computed, hash) != 1 {
		return false, nil
	}
	return true, nil
}

func parseArgon2IDEncoded(encoded string) (salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return nil, nil, ErrInvalidHash
	}
	var mem, timeN, threads int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &timeN, &threads); err != nil {
		return nil, nil, ErrInvalidHash
	}
	if mem != argon2MemoryKiB || timeN != argon2Time || threads != argon2Threads {
		// Reject hashes produced with different parameters (future upgrades must re-hash on login).
		return nil, nil, ErrInvalidHash
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != argon2SaltLen {
		return nil, nil, ErrInvalidHash
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) != argon2KeyLen {
		return nil, nil, ErrInvalidHash
	}
	return salt, hash, nil
}

// PasswordPolicySummary returns non-secret tuning metadata for logs/docs (no hash material).
func PasswordPolicySummary() string {
	return fmt.Sprintf("argon2id m=%dKiB t=%d p=%d key=%dB salt=%dB minLen=%d",
		argon2MemoryKiB/1024, argon2Time, argon2Threads, argon2KeyLen, argon2SaltLen, passwordMinLen)
}

// ValidatePasswordLength returns a client-safe validation error if out of bounds.
func ValidatePasswordLength(password string) error {
	if len(password) < passwordMinLen {
		return ErrPasswordTooShort
	}
	if len(password) > passwordMaxLen {
		return ErrPasswordTooLong
	}
	return nil
}

// Argon2ParamsForTests exposes params for test vectors only.
func Argon2ParamsForTests() (m, t, p, keyLen int) {
	return argon2MemoryKiB, argon2Time, argon2Threads, argon2KeyLen
}

// ParseArgon2MemoryFromEncoded extracts memory KiB from stored hash (for migration tooling); returns 0 if parse fails.
func ParseArgon2MemoryFromEncoded(encoded string) int {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return 0
	}
	var mem, timeN, threads int
	if n, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &timeN, &threads); n != 3 || err != nil {
		return 0
	}
	return mem
}
