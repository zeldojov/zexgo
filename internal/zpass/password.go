package zpass

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      = 64 * 1024 // 64 MiB
	argon2Iterations  = 3
	argon2Parallelism = 2
	argon2SaltLength  = 16
	argon2KeyLength   = 32
)

var (
	errPasswordTooShort       = errors.New("password must be at least 12 characters")
	errPasswordTooLong        = errors.New("password must be at most 128 characters")
	errPasswordInvalidChars   = errors.New("password contains invalid characters")
	errPasswordMissingUpper   = errors.New("password must contain an uppercase letter")
	errPasswordMissingLower   = errors.New("password must contain a lowercase letter")
	errPasswordMissingDigit   = errors.New("password must contain a digit")
	errPasswordMissingSpecial = errors.New("password must contain a special character")

	randRead = rand.Read
)

type PasswordHash string

func NewPasswordHash(password string) (PasswordHash, error) {
	salt := make([]byte, argon2SaltLength)

	if _, err := randRead(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Parallelism,
		argon2KeyLength,
	)

	return PasswordHash(fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)), nil
}

func (h PasswordHash) Verify(password string) bool {
	parts := strings.Split(string(h), "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	var memory, iterations uint32
	var parallelism uint8

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&iterations,
		&parallelism,
	); err != nil {
		return false
	}

	if memory != argon2Memory ||
		iterations != argon2Iterations ||
		parallelism != argon2Parallelism {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	if len(salt) != argon2SaltLength {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	if len(expectedHash) != argon2KeyLength {
		return false
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		argon2KeyLength,
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}
