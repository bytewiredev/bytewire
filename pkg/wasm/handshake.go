//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

const (
	clientProtocolMajor byte = 1
	clientProtocolMinor byte = 0
)

// handleHello processes an OpHello frame from the server.
// It validates the major version matches and sends OpClientHello in response.
func handleHello(data []byte) {
	if len(data) < 3 {
		fmt.Println("bytewire: malformed hello frame")
		return
	}
	serverMajor := data[1]
	serverMinor := data[2]

	fmt.Printf("bytewire: server protocol v%d.%d\n", serverMajor, serverMinor)

	if serverMajor != clientProtocolMajor {
		msg := fmt.Sprintf("Protocol version mismatch: server v%d.%d, client v%d.%d",
			serverMajor, serverMinor, clientProtocolMajor, clientProtocolMinor)
		showErrorOverlay(msg)
		return
	}

	sendClientHello()
}

// sendClientHello sends an OpClientHello frame with the client's protocol version.
func sendClientHello() {
	frameLen := 3 // opcode + major + minor
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x12 // OpClientHello
	frame[5] = clientProtocolMajor
	frame[6] = clientProtocolMinor

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)

	conn.send(uint8Array)
	fmt.Printf("bytewire: sent client hello v%d.%d\n", clientProtocolMajor, clientProtocolMinor)
}
