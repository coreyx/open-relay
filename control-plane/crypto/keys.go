package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// KeyManager manages the Ed25519 cryptographic keys used for signing CWT tokens.
type KeyManager struct {
	KeyID           string
	PrivateKey      ed25519.PrivateKey
	PublicKey       ed25519.PublicKey
	PublicKeyBase64 string
}

// InitKeyManager initializes or loads an Ed25519 keypair.
// It checks ED25519_PRIVATE_KEY env var, then dataDir/ed25519.key, or creates a fresh keypair.
func InitKeyManager(dataDir string) (*KeyManager, error) {
	keyID := os.Getenv("ED25519_KEY_ID")
	if strings.TrimSpace(keyID) == "" {
		keyID = "openrelay_2026"
	}

	var privKey ed25519.PrivateKey

	// 1. Check environment variable
	if envKey := os.Getenv("ED25519_PRIVATE_KEY"); strings.TrimSpace(envKey) != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(envKey))
		if err != nil {
			// Fallback to RawURLEncoding
			raw, err = base64.RawURLEncoding.DecodeString(strings.TrimSpace(envKey))
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode ED25519_PRIVATE_KEY env: %w", err)
		}
		if len(raw) == ed25519.PrivateKeySize {
			privKey = raw
		} else if len(raw) == ed25519.SeedSize {
			privKey = ed25519.NewKeyFromSeed(raw)
		} else {
			return nil, fmt.Errorf("invalid ED25519_PRIVATE_KEY byte length: %d (expected 32 or 64)", len(raw))
		}
	} else {
		// 2. Check key file in dataDir
		keyPath := filepath.Join(dataDir, "ed25519.key")
		if data, err := os.ReadFile(keyPath); err == nil && len(data) == ed25519.PrivateKeySize {
			privKey = ed25519.PrivateKey(data)
		} else {
			// 3. Generate a new keypair
			if err := os.MkdirAll(dataDir, 0700); err != nil {
				return nil, fmt.Errorf("failed to create data dir for key: %w", err)
			}
			_, newPriv, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
			}
			privKey = newPriv
			if err := os.WriteFile(keyPath, privKey, 0600); err != nil {
				return nil, fmt.Errorf("failed to save generated ed25519 key: %w", err)
			}
		}
	}

	pubKey := privKey.Public().(ed25519.PublicKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	km := &KeyManager{
		KeyID:           keyID,
		PrivateKey:      privKey,
		PublicKey:       pubKey,
		PublicKeyBase64: pubKeyB64,
	}

	return km, nil
}
