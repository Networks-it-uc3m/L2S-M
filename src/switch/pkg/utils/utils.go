package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GenerateInterfaceName(prefix, uniqueIdentifier string) (string, error) {
	// Create a SHA-256 hash of the input string
	hash := sha256.New()
	_, err := hash.Write([]byte(uniqueIdentifier))
	if err != nil {
		return "", err
	}
	// Get the full hashed value in hex format
	fullHash := hex.EncodeToString(hash.Sum(nil))

	// Truncate to the first 5 characters of the hash
	digestedName := fullHash[:5]

	// Return the formatted bridge name
	return fmt.Sprintf("%s%s", prefix, digestedName), nil
}
