package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {

	// Generate Alice RSA keys Of 2048 Buts
	alicePrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println(err.Error)
		os.Exit(1)
	}
	// Extract Public Key from RSA Private Key
	alicePublicKey := alicePrivateKey.PublicKey
	secretMessage := "IHsKICAgICAgInByb3ZpZGVyIjogewogICAgICAgICJuYW1lIjogInVjM20iLAogICAgICAgICJkb21haW4iOiAiaWRjby51YzNtLmVzIgogICAgICB9LAogICAgICAiYWNjZXNzTGlzdCI6IFsicHVibGljLWtleS0xIiwgInB1YmxpYy1rZXktMiJdCiAgICB9Cg"
	fmt.Println("Original Text  ", secretMessage)
	signature := SignPKCS1v15(secretMessage, *alicePrivateKey)
	fmt.Println("Singature :  ", signature)
	verif := VerifyPKCS1v15(signature, secretMessage, alicePublicKey)
	fmt.Println(verif)
}

func SignPKCS1v15(plaintext string, privKey rsa.PrivateKey) string {
	// crypto/rand.Reader is a good source of entropy for blinding the RSA
	// operation.
	rng := rand.Reader
	hashed := sha256.Sum256([]byte(plaintext))
	signature, err := rsa.SignPKCS1v15(rng, &privKey, crypto.SHA256, hashed[:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from signing: %s\n", err)
		return "Error from signing"
	}
	return base64.StdEncoding.EncodeToString(signature)
}

func VerifyPKCS1v15(signature string, plaintext string, pubkey rsa.PublicKey) string {
	sig, _ := base64.StdEncoding.DecodeString(signature)
	hashed := sha256.Sum256([]byte(plaintext))
	err := rsa.VerifyPKCS1v15(&pubkey, crypto.SHA256, hashed[:], sig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error from verification: %s\n", err)
		return "Error from verification:"
	}
	return "Signature Verification Passed"
}
