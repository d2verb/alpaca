// Command sign creates an Ed25519 signature for a file.
// Usage: go run tools/sign/main.go <key.pem> <file>
// Output: <file>.sig
package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: sign <key.pem> <file>\n")
		os.Exit(1)
	}

	keyPEM, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read key: %v\n", err)
		os.Exit(1)
	}

	priv, err := parsePrivateKey(keyPEM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse key: %v\n", err)
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	sig := ed25519.Sign(priv, data)
	if err := os.WriteFile(os.Args[2]+".sig", sig, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write signature: %v\n", err)
		os.Exit(1)
	}
}

// parsePrivateKey extracts an Ed25519 private key from PEM data.
// Supports both PKCS#8 DER format and raw 32-byte seed format.
func parsePrivateKey(pemData []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM")
	}

	// Try PKCS#8 first (standard format from openssl/Go)
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		priv, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PEM contains %T, expected Ed25519 key", key)
		}
		return priv, nil
	}

	// Fall back to raw 32-byte seed (from genkey.go)
	if len(block.Bytes) == ed25519.SeedSize {
		return ed25519.NewKeyFromSeed(block.Bytes), nil
	}

	return nil, fmt.Errorf("unsupported key format (PKCS#8 parse failed: %v, raw seed requires %d bytes, got %d)", err, ed25519.SeedSize, len(block.Bytes))
}
