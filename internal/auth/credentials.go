package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"math/big"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP Recommended Settings
const (
	ArgonTime      = 1         // Iterations
	ArgonMemory    = 64 * 1024 // 64 MiB
	ArgonParallel  = 4         // Threads
	ArgonKeyLength = 32        // Output hash size
	ArgonSaltSize  = 16        // Salt size (bytes)

	genPasswordLength = 24
)

// GenerateHash generates a hash for the given secret using the Argon2id algorithm.
// It returns the hash in the format "salt$hash", both base64 encoded.
func GenerateHash(secret string) (string, error) {
	salt := make([]byte, ArgonSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(secret), salt, ArgonTime, ArgonMemory, ArgonParallel, ArgonKeyLength)

	return strings.Join([]string{
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(hash),
	}, "$"), nil
}

// ValidateCreds validates the provided secret against the stored hash.
// The stored hash is expected to be in the format "salt$hash", both base64
// encoded.
func ValidateCreds(secret, stored string) error {
	parts := strings.Split(stored, "$")
	if len(parts) != 2 {
		return errors.New("invalid hash format")
	}

	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return errors.New("invalid salt encoding")
	}

	expectedHash, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.New("invalid hash encoding")
	}

	computedHash := argon2.IDKey([]byte(secret), salt, ArgonTime, ArgonMemory, ArgonParallel, ArgonKeyLength)

	// Constant-time comparison
	if len(computedHash) != len(expectedHash) {
		return errors.New("invalid credentials")
	}
	if subtle.ConstantTimeCompare(computedHash, expectedHash) != 1 {
		return errors.New("invalid credentials")
	}

	return nil
}

// GenPassword generates a random password and its hash.
// It returns the generated password, its hash, and an error if any.
// This function is mostly here to generate a random password for
// an admin user if none exists in the DB upon service startup.
func GenPassword() (string, string, error) {
	psswd := make([]byte, genPasswordLength)
	for i := range psswd {
		n, err := rand.Int(rand.Reader, big.NewInt(95))
		if err != nil {
			return "", "", err
		}
		psswd[i] = byte(n.Int64() + 32)
	}
	hash, err := GenerateHash(string(psswd))
	if err != nil {
		return "", "", err
	}
	return string(psswd), hash, nil
}
