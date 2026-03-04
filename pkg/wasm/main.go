//go:build js && wasm

// Package wasm is the CBS browser client. It receives binary opcodes
// over WebTransport and patches the DOM directly via syscall/js.
package wasm

import (
	"fmt"
	"syscall/js"
)

var (
	document js.Value
	root     js.Value
	nodes    map[uint32]js.Value
)

func init() {
	document = js.Global().Get("document")
	nodes = make(map[uint32]js.Value)
}

// Start initializes the WASM client and connects to the CBS WebTransport endpoint.
func Start() {
	root = document.Call("getElementById", "cbs-root")
	if root.IsNull() {
		fmt.Println("cbs: #cbs-root not found")
		return
	}

	fmt.Println("cbs: WASM client initialized")

	// Register JS exports for the bootstrap script
	js.Global().Set("__cbs_patch", js.FuncOf(patchFromJS))

	// Keep alive
	select {}
}

// patchFromJS is called from JavaScript with a Uint8Array of binary opcodes.
func patchFromJS(_ js.Value, args []js.Value) any {
	if len(args) < 1 {
		return nil
	}
	uint8Array := args[0]
	length := uint8Array.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, uint8Array)

	applyOpcodes(data)
	return nil
}
