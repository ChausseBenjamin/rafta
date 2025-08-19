// sec handles miscelaneous security functions like hash validation and
// password generation.
package sec

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"math/big"
	"runtime"

	"golang.org/x/crypto/argon2"
)

// OWASP Recommended Settings
const (
	ArgonTime      = 1         // Iterations
	ArgonMemory    = 64 * 1024 // 64 MiB
	ArgonKeyLength = 32        // Output hash size
	ArgonSaltSize  = 16        // Salt size (bytes)

	genPasswordLength = 24
)

var (
	threads uint8 = uint8(runtime.NumCPU())

	ErrInvalidCreds     = errors.New("invalid credentials")
	errHashSizeMismatch = errors.New("invalid hash size")
	errInvalidSaltEnc   = errors.New("invalid salt encoding")
	errInvalidHashEnc   = errors.New("invalid hash encoding")
)

// GenerateHash generates a hash for the given secret using the Argon2id algorithm.
// It returns (hash, salt, error)
func GenerateHash(secret string) (string, string, error) {
	return GenerateHashWithThreads(secret, uint8(runtime.NumCPU()))
}

// GenerateHashWithThreads generates a hash for the given secret using the Argon2id algorithm
// with a specific thread count. It returns (hash, salt, error)
func GenerateHashWithThreads(secret string, threads uint8) (string, string, error) {
	salt := make([]byte, ArgonSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}

	hash := argon2.IDKey([]byte(secret), salt, ArgonTime, ArgonMemory, threads, ArgonKeyLength)

	return base64.StdEncoding.EncodeToString(hash),
		base64.StdEncoding.EncodeToString(salt),
		nil
}

// ValidateCreds validates the provided secret against the stored hash.
// The stored hash is expected to be in the format "salt$hash", both base64
// encoded.
func ValidateCreds(passwd, hashStr, saltStr string) error {
	return ValidateCredsWithThreads(passwd, hashStr, saltStr, uint8(runtime.NumCPU()))
}

// ValidateCredsWithThreads validates the provided secret against the stored hash
// with a specific thread count.
func ValidateCredsWithThreads(passwd, hashStr, saltStr string, threads uint8) error {
	dSalt, err := base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return errInvalidSaltEnc
	}

	expHash, err := base64.StdEncoding.DecodeString(hashStr)
	if err != nil {
		return errInvalidHashEnc
	}

	compHash := argon2.IDKey([]byte(passwd), dSalt, ArgonTime, ArgonMemory, threads, ArgonKeyLength)

	// Constant-time comparison
	if len(compHash) != len(expHash) {
		return errHashSizeMismatch
	}
	if subtle.ConstantTimeCompare(compHash, expHash) != 1 {
		return ErrInvalidCreds
	}

	return nil
}

// GenPassword generates a random password and its hash.
// It returns the (passwd, hash, salt, error).
// This function is mostly here to generate a random password for
// an admin user if none exists in the DB upon service startup.
func GenPassword() (string, string, string, error) {
	psswd := make([]byte, genPasswordLength)
	for i := range psswd {
		n, err := rand.Int(rand.Reader, big.NewInt(95))
		if err != nil {
			return "", "", "", err
		}
		psswd[i] = byte(n.Int64() + 32)
	}
	hash, salt, err := GenerateHashWithThreads(string(psswd), uint8(runtime.NumCPU()))
	if err != nil {
		return "", "", "", err
	}
	return string(psswd), hash, salt, nil
}
