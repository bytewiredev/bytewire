//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

// CompletePasskey processes an OpAuthChallenge frame: extracts rpID and challenge,
// calls navigator.credentials.get(), and sends the assertion as OpClientAuth.
func CompletePasskey(data []byte) {
	if len(data) < 2 {
		fmt.Println("bytewire: malformed auth challenge")
		return
	}

	rpIDLen := int(data[1])
	if len(data) < 2+rpIDLen+32 {
		fmt.Println("bytewire: auth challenge too short")
		return
	}
	rpID := string(data[2 : 2+rpIDLen])
	challenge := data[2+rpIDLen : 2+rpIDLen+32]

	go func() {
		navigator := js.Global().Get("navigator")
		if navigator.IsUndefined() {
			fmt.Println("bytewire: navigator not available")
			return
		}
		credentials := navigator.Get("credentials")
		if credentials.IsUndefined() {
			fmt.Println("bytewire: WebAuthn not supported")
			return
		}

		// Build PublicKeyCredentialRequestOptions
		challengeArray := js.Global().Get("Uint8Array").New(len(challenge))
		js.CopyBytesToJS(challengeArray, challenge)

		publicKey := js.Global().Get("Object").New()
		publicKey.Set("challenge", challengeArray)
		publicKey.Set("rpId", rpID)
		publicKey.Set("userVerification", "preferred")

		options := js.Global().Get("Object").New()
		options.Set("publicKey", publicKey)

		// Await credential assertion
		done := make(chan struct{})
		var cred js.Value
		var credErr string

		credentials.Call("get", options).Call("then",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				cred = args[0]
				close(done)
				return nil
			}),
		).Call("catch",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				if len(args) > 0 {
					credErr = args[0].Call("toString").String()
				} else {
					credErr = "unknown error"
				}
				close(done)
				return nil
			}),
		)

		<-done

		if credErr != "" {
			fmt.Println("bytewire: passkey assertion failed:", credErr)
			return
		}

		// Extract assertion data
		response := cred.Get("response")
		credID := extractBytes(cred, "rawId")
		authData := extractResponseBytes(response, "authenticatorData")
		clientDataJSON := extractResponseBytes(response, "clientDataJSON")
		sig := extractResponseBytes(response, "signature")

		sendClientAuth(credID, authData, clientDataJSON, sig)
	}()
}

// extractBytes gets an ArrayBuffer property and converts to Go bytes.
func extractBytes(obj js.Value, prop string) []byte {
	ab := obj.Get(prop)
	if ab.IsUndefined() || ab.IsNull() {
		return nil
	}
	arr := js.Global().Get("Uint8Array").New(ab)
	data := make([]byte, arr.Get("byteLength").Int())
	js.CopyBytesToGo(data, arr)
	return data
}

// extractResponseBytes gets a property from an AuthenticatorAssertionResponse as bytes.
func extractResponseBytes(response js.Value, prop string) []byte {
	ab := response.Get(prop)
	if ab.IsUndefined() || ab.IsNull() {
		return nil
	}
	arr := js.Global().Get("Uint8Array").New(ab)
	data := make([]byte, arr.Get("byteLength").Int())
	js.CopyBytesToGo(data, arr)
	return data
}

// sendClientAuth encodes and sends an OpClientAuth frame.
// Format: [0x13][2B credID len][credID][2B authData len][authData][2B clientDataJSON len][clientDataJSON][2B sig len][sig]
func sendClientAuth(credID, authData, clientDataJSON, sig []byte) {
	payloadLen := 1 + 2 + len(credID) + 2 + len(authData) + 2 + len(clientDataJSON) + 2 + len(sig)
	frame := make([]byte, 4+payloadLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(payloadLen))
	pos := 4
	frame[pos] = 0x13 // OpClientAuth
	pos++
	binary.BigEndian.PutUint16(frame[pos:], uint16(len(credID)))
	pos += 2
	copy(frame[pos:], credID)
	pos += len(credID)
	binary.BigEndian.PutUint16(frame[pos:], uint16(len(authData)))
	pos += 2
	copy(frame[pos:], authData)
	pos += len(authData)
	binary.BigEndian.PutUint16(frame[pos:], uint16(len(clientDataJSON)))
	pos += 2
	copy(frame[pos:], clientDataJSON)
	pos += len(clientDataJSON)
	binary.BigEndian.PutUint16(frame[pos:], uint16(len(sig)))
	pos += 2
	copy(frame[pos:], sig)

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)
	conn.send(uint8Array)

	fmt.Println("bytewire: sent client auth assertion")
}
