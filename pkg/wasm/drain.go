//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

// drainOfflineQueue sends all queued frames over the active connection.
func drainOfflineQueue() {
	frames := GlobalQueue.Flush()
	if len(frames) == 0 {
		return
	}

	fmt.Printf("bytewire: draining %d offline frames\n", len(frames))
	for _, frame := range frames {
		uint8Array := js.Global().Get("Uint8Array").New(len(frame))
		js.CopyBytesToJS(uint8Array, frame)
		conn.send(uint8Array)
	}
}
