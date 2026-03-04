package webauthn

// Credential represents a stored WebAuthn credential.
type Credential struct {
	ID        []byte // credential ID
	PublicKey []byte // COSE-encoded P-256 public key
	UserID    []byte // user identifier
}

// AssertionRequest contains the server's challenge parameters.
type AssertionRequest struct {
	RPID      string
	Challenge []byte
}

// AssertionResponse contains the client's assertion data.
type AssertionResponse struct {
	CredentialID      []byte
	AuthenticatorData []byte
	ClientDataJSON    []byte
	Signature         []byte
}

// VerifyResult holds the outcome of assertion verification.
type VerifyResult struct {
	Success      bool
	CredentialID []byte
}
