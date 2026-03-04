package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"errors"
	"math/big"
)

var (
	ErrInvalidCOSEKey = errors.New("webauthn: invalid COSE key")
	ErrInvalidSig     = errors.New("webauthn: invalid signature")
)

// VerifyAssertion verifies a WebAuthn assertion response against a stored credential.
func VerifyAssertion(req AssertionRequest, resp AssertionResponse, cred Credential) (*VerifyResult, error) {
	x, y, err := parseCOSEP256(cred.PublicKey)
	if err != nil {
		return nil, err
	}

	pub := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	// Compute verification data: authenticatorData || sha256(clientDataJSON)
	clientDataHash := sha256.Sum256(resp.ClientDataJSON)
	verifyData := make([]byte, len(resp.AuthenticatorData)+32)
	copy(verifyData, resp.AuthenticatorData)
	copy(verifyData[len(resp.AuthenticatorData):], clientDataHash[:])

	hash := sha256.Sum256(verifyData)

	if !ecdsa.VerifyASN1(pub, hash[:], resp.Signature) {
		return &VerifyResult{Success: false, CredentialID: resp.CredentialID}, ErrInvalidSig
	}

	return &VerifyResult{Success: true, CredentialID: resp.CredentialID}, nil
}

// parseCOSEP256 extracts x,y coordinates from a COSE-encoded P-256 key.
// Minimal inline CBOR parser -- just enough to extract the COSE map fields.
//
// COSE P-256 key map structure:
//
//	 1 (kty): 2 (EC2)
//	 3 (alg): -7 (ES256)
//	-1 (crv): 1 (P-256)
//	-2 (x): 32 bytes
//	-3 (y): 32 bytes
func parseCOSEP256(data []byte) (*big.Int, *big.Int, error) {
	if len(data) < 10 {
		return nil, nil, ErrInvalidCOSEKey
	}

	var x, y []byte
	pos := 0

	// Read CBOR map header
	if pos >= len(data) {
		return nil, nil, ErrInvalidCOSEKey
	}
	major := data[pos] >> 5
	additional := data[pos] & 0x1f
	pos++
	if major != 5 { // major type 5 = map
		return nil, nil, ErrInvalidCOSEKey
	}

	mapLen := int(additional)
	if additional == 24 {
		if pos >= len(data) {
			return nil, nil, ErrInvalidCOSEKey
		}
		mapLen = int(data[pos])
		pos++
	}

	for i := 0; i < mapLen && pos < len(data); i++ {
		key, n := decodeCBORInt(data[pos:])
		if n == 0 {
			return nil, nil, ErrInvalidCOSEKey
		}
		pos += n

		switch key {
		case -2: // x coordinate
			x, n = decodeCBORBytes(data[pos:])
			if n == 0 {
				return nil, nil, ErrInvalidCOSEKey
			}
			pos += n
		case -3: // y coordinate
			y, n = decodeCBORBytes(data[pos:])
			if n == 0 {
				return nil, nil, ErrInvalidCOSEKey
			}
			pos += n
		default:
			n = skipCBORValue(data[pos:])
			if n == 0 {
				return nil, nil, ErrInvalidCOSEKey
			}
			pos += n
		}
	}

	if len(x) != 32 || len(y) != 32 {
		return nil, nil, ErrInvalidCOSEKey
	}

	return new(big.Int).SetBytes(x), new(big.Int).SetBytes(y), nil
}

// decodeCBORInt reads a CBOR integer (positive or negative).
// Returns the integer value and bytes consumed.
func decodeCBORInt(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}
	major := data[0] >> 5
	additional := data[0] & 0x1f

	var val uint64
	consumed := 1
	if additional < 24 {
		val = uint64(additional)
	} else if additional == 24 {
		if len(data) < 2 {
			return 0, 0
		}
		val = uint64(data[1])
		consumed = 2
	} else {
		return 0, 0
	}

	if major == 0 { // positive integer
		return int(val), consumed
	} else if major == 1 { // negative integer
		return -1 - int(val), consumed
	}
	return 0, 0
}

// decodeCBORBytes reads a CBOR byte string and returns the bytes and total consumed.
func decodeCBORBytes(data []byte) ([]byte, int) {
	if len(data) == 0 {
		return nil, 0
	}
	major := data[0] >> 5
	if major != 2 { // major type 2 = byte string
		return nil, 0
	}
	additional := data[0] & 0x1f
	pos := 1

	var length int
	if additional < 24 {
		length = int(additional)
	} else if additional == 24 {
		if pos >= len(data) {
			return nil, 0
		}
		length = int(data[pos])
		pos++
	} else if additional == 25 {
		if pos+2 > len(data) {
			return nil, 0
		}
		length = int(data[pos])<<8 | int(data[pos+1])
		pos += 2
	} else {
		return nil, 0
	}

	if pos+length > len(data) {
		return nil, 0
	}
	return data[pos : pos+length], pos + length
}

// skipCBORValue skips a CBOR value and returns bytes consumed.
func skipCBORValue(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	major := data[0] >> 5
	additional := data[0] & 0x1f

	switch major {
	case 0, 1: // integer
		if additional < 24 {
			return 1
		} else if additional == 24 {
			return 2
		} else if additional == 25 {
			return 3
		} else if additional == 26 {
			return 5
		}
		return 0
	case 2, 3: // byte/text string
		_, n := decodeCBORBytes(data)
		return n
	default:
		return 1
	}
}
