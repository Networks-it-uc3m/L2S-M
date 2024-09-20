package ovs

import (
	"crypto/sha256"
	"encoding/hex"
)

// generateDatapathID generates a datapath ID from the switch name
func GenerateDatapathID(switchName string) string {
	// Create a new SHA256 hash object
	hash := sha256.New()

	// Write the switch name to the hash object
	hash.Write([]byte(switchName))

	// Get the hashed bytes
	hashedBytes := hash.Sum(nil)

	// Take the first 8 bytes of the hash to create a 64-bit ID
	dpidBytes := hashedBytes[:8]

	// Convert the bytes to a hexadecimal string
	dpid := hex.EncodeToString(dpidBytes)

	return dpid
}
