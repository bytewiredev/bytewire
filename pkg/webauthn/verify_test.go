package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"testing"
)

// buildCOSEP256Key encodes a P-256 public key as a COSE key map.
// CBOR map with 5 entries: {1: 2, 3: -7, -1: 1, -2: x, -3: y}
func buildCOSEP256Key(x, y []byte) []byte {
	var b []byte
	b = append(b, 0xa5) // map of 5 items
	// 1: 2 (kty: EC2)
	b = append(b, 0x01, 0x02)
	// 3: -7 (alg: ES256)
	b = append(b, 0x03, 0x26)
	// -1: 1 (crv: P-256)
	b = append(b, 0x20, 0x01)
	// -2: x (32 bytes)
	b = append(b, 0x21, 0x58, 0x20)
	b = append(b, x...)
	// -3: y (32 bytes)
	b = append(b, 0x22, 0x58, 0x20)
	b = append(b, y...)
	return b
}

func TestVerifyAssertion(t *testing.T) {
	// Generate a test P-256 key pair
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	xBytes := priv.PublicKey.X.FillBytes(make([]byte, 32))
	yBytes := priv.PublicKey.Y.FillBytes(make([]byte, 32))
	coseKey := buildCOSEP256Key(xBytes, yBytes)

	// Create test authenticator data (37 bytes: 32B rpIdHash + 1B flags + 4B signCount)
	rpIDHash := sha256.Sum256([]byte("example.com"))
	authData := make([]byte, 37)
	copy(authData, rpIDHash[:])
	authData[32] = 0x01 // flags: UP=1
	// signCount = 0 (bytes 33-36 are zero)

	clientDataJSON := []byte(`{"type":"webauthn.get","challenge":"dGVzdC1jaGFsbGVuZ2U","origin":"https://example.com"}`)

	// Sign: sha256(authenticatorData || sha256(clientDataJSON))
	clientDataHash := sha256.Sum256(clientDataJSON)
	verifyData := make([]byte, len(authData)+32)
	copy(verifyData, authData)
	copy(verifyData[len(authData):], clientDataHash[:])
	hash := sha256.Sum256(verifyData)

	sig, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		t.Fatal(err)
	}

	challenge, err := NewChallenge()
	if err != nil {
		t.Fatal(err)
	}

	cred := Credential{
		ID:        []byte("cred-id-1"),
		PublicKey: coseKey,
		UserID:    []byte("user-1"),
	}

	tests := []struct {
		name    string
		cred    Credential
		sig     []byte
		wantOK  bool
		wantErr error
	}{
		{
			name:   "valid assertion",
			cred:   cred,
			sig:    sig,
			wantOK: true,
		},
		{
			name:    "invalid signature",
			cred:    cred,
			sig:     []byte("bad-signature"),
			wantOK:  false,
			wantErr: ErrInvalidSig,
		},
		{
			name: "corrupt COSE key",
			cred: Credential{
				ID:        []byte("cred-id-2"),
				PublicKey: []byte{0xff, 0xff, 0xff},
				UserID:    []byte("user-2"),
			},
			sig:     sig,
			wantOK:  false,
			wantErr: ErrInvalidCOSEKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := AssertionRequest{
				RPID:      "example.com",
				Challenge: challenge,
			}
			resp := AssertionResponse{
				CredentialID:      tt.cred.ID,
				AuthenticatorData: authData,
				ClientDataJSON:    clientDataJSON,
				Signature:         tt.sig,
			}

			result, err := VerifyAssertion(req, resp, tt.cred)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}

			if tt.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil || !result.Success {
					t.Fatal("expected success=true")
				}
			} else if result != nil && result.Success {
				t.Fatal("expected success=false")
			}
		})
	}
}
