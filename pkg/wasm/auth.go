//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

// RequestPasskey triggers a WebAuthn credential assertion in the browser.
// This is used for tokenless authentication via hardware passkeys.
func RequestPasskey(rpID string, challenge []byte) (js.Value, error) {
	navigator := js.Global().Get("navigator")
	if navigator.IsUndefined() {
		return js.Undefined(), fmt.Errorf("cbs: navigator not available")
	}

	credentials := navigator.Get("credentials")
	if credentials.IsUndefined() {
		return js.Undefined(), fmt.Errorf("cbs: WebAuthn not supported")
	}

	// Build the PublicKeyCredentialRequestOptions
	challengeArray := js.Global().Get("Uint8Array").New(len(challenge))
	js.CopyBytesToJS(challengeArray, challenge)

	options := js.Global().Get("Object").New()
	publicKey := js.Global().Get("Object").New()
	publicKey.Set("challenge", challengeArray)
	publicKey.Set("rpId", rpID)
	publicKey.Set("userVerification", "preferred")
	options.Set("publicKey", publicKey)

	// Returns a Promise that resolves to a PublicKeyCredential
	promise := credentials.Call("get", options)
	return promise, nil
}
