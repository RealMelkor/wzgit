package db

import (
	"errors"
	"strconv"
	"unicode"
	"fmt"
	"strings"
	"crypto/rand"
	"crypto/subtle"
        "encoding/base64"

	"golang.org/x/crypto/argon2"
	passwordvalidator "github.com/wagslane/go-password-validator"
)

const passwordTime = 1
const passwordMemory = 64 * 1024
const passwordThreads = 4
const passwordKeyLen = 32

func hashPassword(password string) (string, error) {

        if err := isPasswordValid(password); err != nil { return "", err }

        // Generate a Salt
        salt := make([]byte, 16)
        if _, err := rand.Read(salt); err != nil {
                return "", err
        }

        hash := argon2.IDKey(
                []byte(password), salt, passwordTime,
                passwordMemory, passwordThreads, passwordKeyLen,
        )

        // Base64 encode the salt and hashed password.
        b64Salt := base64.RawStdEncoding.EncodeToString(salt)
        b64Hash := base64.RawStdEncoding.EncodeToString(hash)

        format := "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s"
        full := fmt.Sprintf(
                format, argon2.Version, passwordMemory,
                passwordTime, passwordThreads, b64Salt, b64Hash,
        )
        return full, nil
}

func checkPassword(password, hash string) error {

        parts := strings.Split(hash, "$")
        if (len(parts) < 4) {
                return errors.New("invalid hash")
        }
        var time uint32
        var memory uint32
        var threads uint8

        _, err := fmt.Sscanf(
                parts[3], "m=%d,t=%d,p=%d",
                &memory, &time, &threads,
        )
        if err != nil {
                return err
        }

        salt, err := base64.RawStdEncoding.DecodeString(parts[4])
        if err != nil {
                return err
        }

        decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
        if err != nil {
                return err
        }
        keyLen := uint32(len(decodedHash))

        comparisonHash := argon2.IDKey(
                []byte(password), salt, time,
                memory, threads, keyLen,
        )

        if subtle.ConstantTimeCompare(decodedHash, comparisonHash) != 1 {
                return errors.New("password doesn't match")
        }

	return nil
}

const (
	passwordMaxLen = 256
	maxNameLen = 24
)

func isPasswordValid(password string) (error) {
	if len(password) == 0 { return errors.New("empty password") }
	if len(password) > passwordMaxLen {
		return errors.New("password too long(maximum " +
				strconv.Itoa(passwordMaxLen) +
				" characters)")
	}
	return passwordvalidator.Validate(password, 50)
}

func isNameValid(name string) error {
	if len(name) == 0 { return errors.New("empty name") }
	if len(name) > maxNameLen { return errors.New("name too long") }
	if !unicode.IsLetter([]rune(name)[0]) {
		return errors.New("your name must start with a letter")
	}
	return nil
}

var blacklisted = map[string]bool{
	"anon": true,
	"root": true,
	"account": true,
	"public": true,
	"login": true,
	"register": true,
	"captcha": true,
	"static": true,
}

func isUsernameValid(name string) error {
	if blacklisted[name] { return errors.New("blacklisted username") }
	if err := isNameValid(name); err != nil { return err }
	for _, c := range name {
		if c > unicode.MaxASCII || (!unicode.IsLetter(c) &&
				!unicode.IsNumber(c)) {
			return errors.New(
				"your name contains invalid characters")
		}
	}
	return nil
}

func isGroupNameValid(name string) (error) {
	if err := isNameValid(name); err != nil { return err }
	for _, c := range name {
		if c > unicode.MaxASCII ||
		   (!unicode.IsLetter(c) && !unicode.IsNumber(c) &&
		   c != '-' && c != '_') {
			return errors.New("the group name " +
					  "contains invalid characters")
		}
	}
	return nil
}

func IsRepoNameValid(name string) (error) {
	if err := isNameValid(name); err != nil { return err }
	for _, c := range name {
		if c > unicode.MaxASCII ||
		   (!unicode.IsLetter(c) && !unicode.IsNumber(c) &&
		   c != '-' && c != '_') {
			return errors.New("the repository name " +
					  "contains invalid characters")
		}
	}
	return nil
}
