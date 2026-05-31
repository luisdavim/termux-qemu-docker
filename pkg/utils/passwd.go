package utils

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
)

func EncryptPassword(userPassword string) (string, error) {
	// Generate a random string for use in the salt
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	s := make([]byte, 8)
	for i := range s {
		s[i] = charset[seededRand.Intn(len(charset))]
	}
	salt := fmt.Appendf(nil, "$6$%s", s)
	// use salt to hash user-supplied password
	c := crypt.SHA512.New()
	hash, err := c.Generate([]byte(userPassword), salt)
	if err != nil {
		return "", fmt.Errorf("error hashing user's supplied password: %w", err)
	}
	return string(hash), nil
}
