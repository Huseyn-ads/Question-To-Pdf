package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func HashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}

func GenerateToken() (string, []byte, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)

	if err != nil {
		return "", nil, fmt.Errorf("generate random session token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(randomBytes)

	return token, HashToken(token), nil

}
