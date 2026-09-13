package crypto

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/veraison/go-cose"
)

func TestCWTGenerationAndVerification(t *testing.T) {
	tempDir := t.TempDir()
	km, err := InitKeyManager(tempDir)
	if err != nil {
		t.Fatalf("InitKeyManager failed: %v", err)
	}

	docID := "test_doc_123"
	userID := "usr_alice"
	aud := "http://localhost:8080"
	channel := "relay_guid_456"

	tokenStr, err := km.SignDocToken(docID, userID, aud, channel, false, 3600)
	if err != nil {
		t.Fatalf("SignDocToken failed: %v", err)
	}

	rawBytes, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		t.Fatalf("Failed to decode token base64: %v", err)
	}

	// Unmarshal CWT Tag 61
	var cwtTag cbor.Tag
	if err := cbor.Unmarshal(rawBytes, &cwtTag); err != nil {
		t.Fatalf("Failed to unmarshal CWT tag 61: %v", err)
	}
	if cwtTag.Number != 61 {
		t.Fatalf("Expected CBOR tag 61, got %d", cwtTag.Number)
	}

	// Content of tag 61 is the COSE Sign1 message
	coseBytes, ok := cwtTag.Content.([]byte)
	if !ok {
		t.Fatalf("Expected CWT content to be []byte, got: %T", cwtTag.Content)
	}

	// Verify with go-cose verifier using public key
	verifier, err := cose.NewVerifier(cose.AlgorithmEd25519, km.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create COSE verifier: %v", err)
	}

	var sign1 cose.Sign1Message
	if err := sign1.UnmarshalCBOR(coseBytes); err != nil {
		t.Fatalf("Failed to unmarshal COSE Sign1 message: %v", err)
	}

	if err := sign1.Verify(nil, verifier); err != nil {
		t.Fatalf("COSE Sign1 signature verification failed: %v", err)
	}

	// Unmarshal claims payload
	var claims CWTClaims
	if err := cbor.Unmarshal(sign1.Payload, &claims); err != nil {
		t.Fatalf("Failed to unmarshal CWT claims: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("Expected sub '%s', got '%s'", userID, claims.Subject)
	}
	if claims.Audience != aud {
		t.Errorf("Expected aud '%s', got '%s'", aud, claims.Audience)
	}
	if claims.Scope != "doc:test_doc_123:rw" {
		t.Errorf("Expected scope 'doc:test_doc_123:rw', got '%s'", claims.Scope)
	}
	if claims.Channel != channel {
		t.Errorf("Expected channel '%s', got '%s'", channel, claims.Channel)
	}
	if claims.Expiration <= time.Now().Unix() {
		t.Errorf("Expected expiration to be in future, got %d", claims.Expiration)
	}
}

func TestFileTokenGeneration(t *testing.T) {
	tempDir := t.TempDir()
	km, err := InitKeyManager(tempDir)
	if err != nil {
		t.Fatalf("InitKeyManager failed: %v", err)
	}

	fileHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	docID := "doc_xyz"
	userID := "usr_bob"
	aud := "https://relay.example.com"
	channel := "relay_ch_1"

	tokenStr, err := km.SignFileToken(docID, fileHash, userID, aud, channel, false, 1800)
	if err != nil {
		t.Fatalf("SignFileToken failed: %v", err)
	}

	rawBytes, err := base64.RawURLEncoding.DecodeString(tokenStr)
	if err != nil {
		t.Fatalf("Failed to decode token base64: %v", err)
	}

	var cwtTag cbor.Tag
	if err := cbor.Unmarshal(rawBytes, &cwtTag); err != nil {
		t.Fatalf("Failed to unmarshal CWT tag 61: %v", err)
	}
	if cwtTag.Number != 61 {
		t.Fatalf("Expected CBOR tag 61, got %d", cwtTag.Number)
	}

	coseBytes, ok := cwtTag.Content.([]byte)
	if !ok {
		t.Fatalf("Expected CWT content to be []byte, got: %T", cwtTag.Content)
	}

	var sign1 cose.Sign1Message
	if err := sign1.UnmarshalCBOR(coseBytes); err != nil {
		t.Fatalf("Failed to unmarshal Sign1: %v", err)
	}

	verifier, err := cose.NewVerifier(cose.AlgorithmEd25519, km.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create verifier: %v", err)
	}

	if err := sign1.Verify(nil, verifier); err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	var claims CWTClaims
	if err := cbor.Unmarshal(sign1.Payload, &claims); err != nil {
		t.Fatalf("Failed to unmarshal claims: %v", err)
	}

	expectedScope := "file:" + fileHash + ":" + docID + ":rw"
	if claims.Scope != expectedScope {
		t.Errorf("Expected scope '%s', got '%s'", expectedScope, claims.Scope)
	}
}
