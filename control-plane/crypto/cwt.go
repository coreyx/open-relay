package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/veraison/go-cose"
)

// CWTClaims represents the standard and custom claims for Relay CWT.
type CWTClaims struct {
	Issuer     string `cbor:"1,keyasint,omitempty"`
	Subject    string `cbor:"2,keyasint,omitempty"`
	Audience   string `cbor:"3,keyasint,omitempty"`
	Expiration int64  `cbor:"4,keyasint,omitempty"`
	IssuedAt   int64  `cbor:"6,keyasint,omitempty"`
	Scope      string `cbor:"-80201,keyasint"`
	Channel    string `cbor:"-80202,keyasint,omitempty"`
}

// SignToken generates an RFC 8392 CWT wrapped in COSE Sign1 (tag 18) and CWT (tag 61).
func (km *KeyManager) SignToken(claims CWTClaims) (string, error) {
	if claims.IssuedAt == 0 {
		claims.IssuedAt = time.Now().Unix()
	}
	if claims.Expiration == 0 {
		claims.Expiration = claims.IssuedAt + 3600 // 1 hour default
	}

	// 1. Serialize claims to CBOR
	claimsBytes, err := cbor.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal CWT claims: %w", err)
	}

	// 2. Build COSE Sign1 protected header (alg: -8 EdDSA, kid)
	headers := cose.Headers{
		Protected: cose.ProtectedHeader{
			cose.HeaderLabelAlgorithm: cose.AlgorithmEd25519,
			cose.HeaderLabelKeyID:     []byte(km.KeyID),
		},
	}

	// 3. Sign using go-cose Signer
	signer, err := cose.NewSigner(cose.AlgorithmEd25519, km.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create COSE signer: %w", err)
	}

	// cose.Sign1 creates the raw COSE_Sign1 message structure
	coseMsg, err := cose.Sign1(rand.Reader, signer, headers, claimsBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to sign COSE message: %w", err)
	}

	// 4. Wrap with CWT CBOR tag 61 (RFC 8392). Note: cose.Sign1 already includes COSE_Sign1 tag 18.
	cwtTag := cbor.Tag{Number: 61, Content: cbor.RawMessage(coseMsg)}
	cwtBytes, err := cbor.Marshal(cwtTag)
	if err != nil {
		return "", fmt.Errorf("failed to tag CWT: %w", err)
	}

	// 5. Encode as base64url string
	tokenB64 := base64.RawURLEncoding.EncodeToString(cwtBytes)
	return tokenB64, nil
}

// SignDocToken signs a document token with given docId, auth, audience, and channel.
func (km *KeyManager) SignDocToken(docID, userID, audience, channel string, isReadOnly bool, validForSeconds int64) (string, error) {
	scopeAuth := "rw"
	if isReadOnly {
		scopeAuth = "r"
	}
	scope := fmt.Sprintf("doc:%s:%s", docID, scopeAuth)

	now := time.Now().Unix()
	exp := now + validForSeconds
	if validForSeconds <= 0 {
		exp = now + 3600
	}

	claims := CWTClaims{
		Subject:    userID,
		Audience:   audience,
		IssuedAt:   now,
		Expiration: exp,
		Scope:      scope,
		Channel:    channel,
	}

	return km.SignToken(claims)
}

// SignFileToken signs a file token with given fileHash, docId, auth, audience, and channel.
func (km *KeyManager) SignFileToken(docID, fileHash, userID, audience, channel string, isReadOnly bool, validForSeconds int64) (string, error) {
	scopeAuth := "rw"
	if isReadOnly {
		scopeAuth = "r"
	}
	scope := fmt.Sprintf("file:%s:%s:%s", fileHash, docID, scopeAuth)

	now := time.Now().Unix()
	exp := now + validForSeconds
	if validForSeconds <= 0 {
		exp = now + 3600
	}

	claims := CWTClaims{
		Subject:    userID,
		Audience:   audience,
		IssuedAt:   now,
		Expiration: exp,
		Scope:      scope,
		Channel:    channel,
	}

	return km.SignToken(claims)
}
