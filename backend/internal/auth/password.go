package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordRunes = 15
	MaxPasswordBytes = 4096

	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 1
	argonSaltBytes   = 16
	argonKeyBytes    = 32
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 15 characters")
	ErrPasswordTooLong  = errors.New("password is too long")
	ErrPasswordInvalid  = errors.New("password is invalid")
)

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltBytes   uint32
	keyBytes    uint32
}

var defaultArgonParams = argonParams{
	memory:      argonMemory,
	iterations:  argonIterations,
	parallelism: argonParallelism,
	saltBytes:   argonSaltBytes,
	keyBytes:    argonKeyBytes,
}

func HashPassword(password string) (string, error) {
	return hashPassword(password, defaultArgonParams)
}

func VerifyPassword(password string, encodedHash string) (bool, error) {
	if err := ValidatePassword(password); err != nil {
		return false, err
	}

	params, salt, expectedHash, err := decodePasswordHash(encodedHash)
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyBytes)
	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}

func ValidatePassword(password string) error {
	if !utf8.ValidString(password) {
		return ErrPasswordInvalid
	}
	if len([]byte(password)) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	if utf8.RuneCountInString(password) < MinPasswordRunes {
		return ErrPasswordTooShort
	}

	return nil
}

func hashPassword(password string, params argonParams) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	salt := make([]byte, params.saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyBytes)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		params.memory,
		params.iterations,
		params.parallelism,
		b64Salt,
		b64Hash,
	), nil
}

func decodePasswordHash(encodedHash string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argonParams{}, nil, nil, ErrPasswordInvalid
	}

	params, err := parseArgonParams(parts[3])
	if err != nil {
		return argonParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, ErrPasswordInvalid
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, ErrPasswordInvalid
	}

	params.saltBytes = uint32(len(salt))
	params.keyBytes = uint32(len(hash))

	return params, salt, hash, nil
}

func parseArgonParams(encoded string) (argonParams, error) {
	values := map[string]string{}
	for _, part := range strings.Split(encoded, ",") {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return argonParams{}, ErrPasswordInvalid
		}
		values[keyValue[0]] = keyValue[1]
	}

	memory, err := strconv.ParseUint(values["m"], 10, 32)
	if err != nil {
		return argonParams{}, ErrPasswordInvalid
	}
	iterations, err := strconv.ParseUint(values["t"], 10, 32)
	if err != nil {
		return argonParams{}, ErrPasswordInvalid
	}
	parallelism, err := strconv.ParseUint(values["p"], 10, 8)
	if err != nil {
		return argonParams{}, ErrPasswordInvalid
	}

	return argonParams{
		memory:      uint32(memory),
		iterations:  uint32(iterations),
		parallelism: uint8(parallelism),
	}, nil
}
