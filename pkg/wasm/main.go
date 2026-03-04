//go:build js && wasm

// Package wasm is the CBS browser client. It connects WebTransport,
// receives binary opcodes, patches the DOM, and delegates events —
// all from Go via syscall/js. No external JavaScript required.
package wasm

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"syscall/js"
)

var (
	document js.Value
	root     js.Value
	nodes    map[uint32]js.Value

	// intentWriter is the persistent WritableStreamDefaultWriter
	// for sending client intents over a single bidi stream.
	intentWriter js.Value
)

func init() {
	document = js.Global().Get("document")
	nodes = make(map[uint32]js.Value)
}

// Start initializes the CBS WASM client: connects WebTransport,
// reads server patches, and sets up event delegation.
func Start() {
	root = document.Call("getElementById", "cbs-root")
	if root.IsNull() {
		fmt.Println("cbs: #cbs-root not found")
		return
	}

	root.Set("textContent", "Connecting…")
	fmt.Println("cbs: WASM client initialized")

	// Read config from window.__cbs_config injected by the server.
	config := js.Global().Get("__cbs_config")
	if config.IsUndefined() || config.IsNull() {
		root.Set("textContent", "Error: __cbs_config not set")
		return
	}

	wtURL := config.Get("url").String()
	certHash := config.Get("certHash")

	// Connect WebTransport
	wtOpts := js.Global().Get("Object").New()
	hashObj := js.Global().Get("Object").New()
	hashObj.Set("algorithm", "sha-256")
	hashObj.Set("value", certHash.Get("buffer"))
	hashes := js.Global().Get("Array").New()
	hashes.Call("push", hashObj)
	wtOpts.Set("serverCertificateHashes", hashes)

	wt := js.Global().Get("WebTransport").New(wtURL, wtOpts)

	// Await wt.ready
	done := make(chan struct{})
	var connectErr string

	wt.Get("ready").Call("then",
		js.FuncOf(func(_ js.Value, _ []js.Value) any {
			close(done)
			return nil
		}),
	).Call("catch",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 {
				connectErr = args[0].Call("toString").String()
			} else {
				connectErr = "unknown error"
			}
			close(done)
			return nil
		}),
	)

	<-done

	if connectErr != "" {
		root.Set("textContent", "WebTransport failed: "+connectErr)
		fmt.Println("cbs: WebTransport failed:", connectErr)
		return
	}

	fmt.Println("cbs: WebTransport connected")
	root.Set("textContent", "")

	// Start reading server patches (unidirectional streams)
	go readIncomingStreams(wt)

	// Open one persistent bidi stream for client→server intents
	openDone := make(chan struct{})
	wt.Call("createBidirectionalStream").Call("then",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			stream := args[0]
			writable := stream.Get("writable")
			intentWriter = writable.Call("getWriter")
			close(openDone)
			return nil
		}),
	).Call("catch",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			fmt.Println("cbs: failed to open intent stream", args[0].Call("toString").String())
			close(openDone)
			return nil
		}),
	)
	<-openDone

	if intentWriter.IsUndefined() || intentWriter.IsNull() {
		root.Set("textContent", "Error: could not open intent stream")
		return
	}

	// Listen for browser back/forward navigation
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		path := js.Global().Get("location").Get("pathname").String()
		sendClientNav(path)
		return nil
	}))

	// Set up event delegation on #cbs-root
	setupEventDelegation()

	fmt.Println("cbs: event delegation active")

	// Keep alive
	select {}
}

// readIncomingStreams reads unidirectional streams from the server
// and applies binary opcode patches to the DOM.
func readIncomingStreams(wt js.Value) {
	reader := wt.Get("incomingUnidirectionalStreams").Call("getReader")

	var readNext js.Func
	readNext = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		reader.Call("read").Call("then",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				result := args[0]
				if result.Get("done").Bool() {
					return nil
				}
				stream := result.Get("value")
				go readStreamAndPatch(stream, reader, readNext)
				return nil
			}),
		).Call("catch",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				fmt.Println("cbs: stream reader error:", args[0].Call("toString").String())
				return nil
			}),
		)
		return nil
	})

	readNext.Invoke()
}

// readStreamAndPatch fully reads a unidirectional stream and applies the opcodes.
func readStreamAndPatch(stream js.Value, reader js.Value, readNext js.Func) {
	sr := stream.Call("getReader")
	var chunks []js.Value

	var readChunk js.Func
	readChunk = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		sr.Call("read").Call("then",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				result := args[0]
				if result.Get("done").Bool() {
					// All chunks read — concatenate and patch
					total := 0
					for _, c := range chunks {
						total += c.Get("byteLength").Int()
					}
					buf := js.Global().Get("Uint8Array").New(total)
					offset := 0
					for _, c := range chunks {
						buf.Call("set", c, offset)
						offset += c.Get("byteLength").Int()
					}

					data := make([]byte, total)
					js.CopyBytesToGo(data, buf)
					applyOpcodes(data)

					// Read next stream
					readNext.Invoke()
					return nil
				}
				chunks = append(chunks, result.Get("value"))
				readChunk.Invoke()
				return nil
			}),
		).Call("catch",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				fmt.Println("cbs: chunk read error:", args[0].Call("toString").String())
				readNext.Invoke()
				return nil
			}),
		)
		return nil
	})

	readChunk.Invoke()
}

// sendIntent encodes and sends an OpClientIntent frame over the persistent bidi stream.
func sendIntent(nodeID uint32, eventType byte, payload []byte) {
	frameLen := 1 + 4 + 1 + len(payload)
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x10 // OpClientIntent
	binary.BigEndian.PutUint32(frame[5:9], nodeID)
	frame[9] = eventType
	copy(frame[10:], payload)

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)

	intentWriter.Call("write", uint8Array).Call("catch",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			fmt.Println("cbs: send intent failed:", args[0].Call("toString").String())
			return nil
		}),
	)
}

// sendClientNav encodes and sends an OpClientNav frame over the persistent bidi stream.
func sendClientNav(path string) {
	frameLen := 1 + len(path)
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x11 // OpClientNav
	copy(frame[5:], path)

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)

	intentWriter.Call("write", uint8Array).Call("catch",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			fmt.Println("cbs: send nav failed:", args[0].Call("toString").String())
			return nil
		}),
	)
}

// findCBSNode walks up from el to find the nearest element with data-cbs-id.
func findCBSNode(el js.Value) (uint32, bool) {
	body := document.Get("body")
	for !el.IsNull() && !el.IsUndefined() && !el.Equal(body) {
		attr := el.Call("getAttribute", "data-cbs-id")
		if !attr.IsNull() && !attr.IsUndefined() {
			id, err := strconv.Atoi(attr.String())
			if err == nil {
				return uint32(id), true
			}
		}
		el = el.Get("parentElement")
	}
	return 0, false
}

// setupEventDelegation attaches event listeners on #cbs-root that
// map DOM events to CBS binary intents.
func setupEventDelegation() {
	// Click
	root.Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")
		if nodeID, ok := findCBSNode(target); ok {
			e.Call("stopPropagation")
			sendIntent(nodeID, 0x01, nil) // EventClick
		}
		return nil
	}))

	// Input
	root.Call("addEventListener", "input", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")
		if nodeID, ok := findCBSNode(target); ok {
			val := target.Get("value").String()
			sendIntent(nodeID, 0x02, []byte(val)) // EventInput
		}
		return nil
	}))

	// Submit
	root.Call("addEventListener", "submit", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		e.Call("preventDefault")
		target := e.Get("target")
		if nodeID, ok := findCBSNode(target); ok {
			sendIntent(nodeID, 0x03, nil) // EventSubmit
		}
		return nil
	}))

	// SPA link interception: <a data-cbs-link href="/path">
	root.Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")

		// Walk up to find nearest <a> with data-cbs-link
		body := document.Get("body")
		el := target
		for !el.IsNull() && !el.IsUndefined() && !el.Equal(body) {
			tagName := el.Get("tagName")
			if !tagName.IsUndefined() && tagName.String() == "A" {
				linkAttr := el.Call("getAttribute", "data-cbs-link")
				if !linkAttr.IsNull() && !linkAttr.IsUndefined() {
					href := el.Call("getAttribute", "href")
					if !href.IsNull() && !href.IsUndefined() {
						e.Call("preventDefault")
						e.Call("stopPropagation")
						sendClientNav(href.String())
					}
					return nil
				}
			}
			el = el.Get("parentElement")
		}
		return nil
	}))
}
