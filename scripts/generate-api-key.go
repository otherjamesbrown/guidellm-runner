// +build ignore

// Script to generate an API key and its fingerprint for seeding the database.
// Usage: go run scripts/generate-api-key.go
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func main() {
	// Generate a 32-byte random secret
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic(err)
	}

	// Create the full token in the format: ai-aas_<key_id>_<secret>
	keyID := base64.RawURLEncoding.EncodeToString(secret[:6]) // Short key ID from first 6 bytes
	secretPart := base64.RawURLEncoding.EncodeToString(secret)
	token := fmt.Sprintf("ai-aas_%s_%s", keyID, secretPart)

	// Compute fingerprint (SHA-256 hash of full token, base64 encoded)
	fingerprintHash := sha256.Sum256([]byte(token))
	fingerprint := base64.RawURLEncoding.EncodeToString(fingerprintHash[:])

	fmt.Println("=== Generated API Key ===")
	fmt.Printf("Token (save this - shown only once):\n%s\n\n", token)
	fmt.Printf("Key ID: %s\n", keyID)
	fmt.Printf("Fingerprint: %s\n\n", fingerprint)

	fmt.Println("=== SQL to insert (update UUIDs as needed) ===")
	fmt.Printf(`INSERT INTO api_keys (
    org_id,
    principal_type,
    principal_id,
    fingerprint,
    status,
    scopes,
    key_id,
    notes
) VALUES (
    'b6fc81af-a245-4599-b3e1-7d2b8745c148',  -- master-admin-org
    'user',
    'b6fc81af-a245-4599-b3e1-7d2b8745c148',  -- same as org for service account
    '%s',
    'active',
    '["*"]',
    'guidellm-benchmark-key',
    'API key for guidellm-runner benchmarks'
);
`, fingerprint)
}
