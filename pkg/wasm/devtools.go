//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

// initDevTools creates the window.__bytewire object for console-based state inspection.
func initDevTools() {
	bw := js.Global().Get("Object").New()
	bw.Set("version", "0.1.0")
	bw.Set("state", js.Null())

	// __bytewire.nodes — getter that returns current node count
	bw.Set("nodes", len(nodes))

	// __bytewire.requestState() — sends a request to server for state snapshot
	bw.Set("requestState", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		sendDevToolsRequest()
		return nil
	}))

	js.Global().Get("window").Set("__bytewire", bw)
	fmt.Println("bytewire: DevTools API available at window.__bytewire")
}

// updateDevToolsNodeCount refreshes the node count on the __bytewire object.
func updateDevToolsNodeCount() {
	bw := js.Global().Get("window").Get("__bytewire")
	if bw.IsUndefined() || bw.IsNull() {
		return
	}
	bw.Set("nodes", len(nodes))
}

// handleDevToolsState processes an OpDevToolsState frame by setting
// window.__bytewire.state to the received JSON string.
func handleDevToolsState(data []byte) {
	if len(data) < 5 {
		return
	}
	jsonLen := int(binary.BigEndian.Uint32(data[1:5]))
	if len(data) < 5+jsonLen {
		return
	}
	jsonStr := string(data[5 : 5+jsonLen])

	bw := js.Global().Get("window").Get("__bytewire")
	if bw.IsUndefined() || bw.IsNull() {
		return
	}
	bw.Set("state", jsonStr)
	fmt.Println("bytewire: DevTools state updated — inspect with window.__bytewire.state")
}

// sendDevToolsRequest sends a minimal client message requesting a state snapshot.
// We reuse the intent stream with a special opcode pattern: OpClientIntent with nodeID=0
// and EventType=0xFF as a sentinel for "devtools request".
func sendDevToolsRequest() {
	frameLen := 1 + 4 + 1
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x10 // OpClientIntent
	binary.BigEndian.PutUint32(frame[5:9], 0)
	frame[9] = 0xFF // sentinel: devtools state request

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)

	conn.send(uint8Array)
}
