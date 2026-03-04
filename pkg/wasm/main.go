//go:build js && wasm

// Package wasm is the Bytewire browser client. It connects WebTransport,
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

	// overlay is the reconnection status overlay element.
	overlay js.Value
)

func init() {
	document = js.Global().Get("document")
	nodes = make(map[uint32]js.Value)
	overlay = js.Undefined()
}

// Start initializes the Bytewire WASM client: connects WebTransport,
// reads server patches, and sets up event delegation.
func Start() {
	root = document.Call("getElementById", "bw-root")
	if root.IsNull() {
		fmt.Println("bytewire: #bw-root not found")
		return
	}

	root.Set("textContent", "Connecting…")
	fmt.Println("bytewire: WASM client initialized")

	// Expose window.__bytewire DevTools object
	initDevTools()

	if !connect() {
		return
	}

	// Listen for browser back/forward navigation
	js.Global().Get("window").Call("addEventListener", "popstate", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		path := js.Global().Get("location").Get("pathname").String()
		sendClientNav(path)
		return nil
	}))

	// Set up event delegation on #bw-root
	setupEventDelegation()

	fmt.Println("bytewire: event delegation active")

	// Keep alive
	select {}
}

// connect establishes the WebTransport connection and starts reading streams.
// Returns true on success, false on fatal config errors.
func connect() bool {
	// Read config from window.__bw_config injected by the server.
	config := js.Global().Get("__bw_config")
	if config.IsUndefined() || config.IsNull() {
		root.Set("textContent", "Error: __bw_config not set")
		return false
	}

	wtURL := config.Get("url").String()
	certHash := config.Get("certHash")

	wt, err := dialWebTransport(wtURL, certHash)
	if err != "" {
		root.Set("textContent", "WebTransport failed: "+err)
		fmt.Println("bytewire: WebTransport failed:", err)
		return false
	}

	fmt.Println("bytewire: WebTransport connected")
	root.Set("textContent", "")

	if !openIntentStream(wt) {
		root.Set("textContent", "Error: could not open intent stream")
		return false
	}

	// Start reading server patches; on disconnect, trigger reconnect.
	go readIncomingStreamsWithReconnect(wt)

	return true
}

// dialWebTransport creates a WebTransport connection and awaits ready.
// Returns the WebTransport object and empty string on success, or
// js.Undefined() and an error string on failure.
func dialWebTransport(wtURL string, certHash js.Value) (js.Value, string) {
	wtOpts := js.Global().Get("Object").New()
	hashObj := js.Global().Get("Object").New()
	hashObj.Set("algorithm", "sha-256")
	hashObj.Set("value", certHash.Get("buffer"))
	hashes := js.Global().Get("Array").New()
	hashes.Call("push", hashObj)
	wtOpts.Set("serverCertificateHashes", hashes)

	wt := js.Global().Get("WebTransport").New(wtURL, wtOpts)

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
	return wt, connectErr
}

// openIntentStream opens the persistent bidi stream for client intents.
func openIntentStream(wt js.Value) bool {
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
			fmt.Println("bytewire: failed to open intent stream", args[0].Call("toString").String())
			close(openDone)
			return nil
		}),
	)
	<-openDone

	return !intentWriter.IsUndefined() && !intentWriter.IsNull()
}

// readIncomingStreamsWithReconnect reads streams and triggers reconnection on disconnect.
func readIncomingStreamsWithReconnect(wt js.Value) {
	// Wait for the WebTransport session to close.
	closed := make(chan struct{})
	wt.Get("closed").Call("then",
		js.FuncOf(func(_ js.Value, _ []js.Value) any {
			close(closed)
			return nil
		}),
	).Call("catch",
		js.FuncOf(func(_ js.Value, _ []js.Value) any {
			close(closed)
			return nil
		}),
	)

	// Read streams normally while connected.
	go readIncomingStreams(wt)

	<-closed
	fmt.Println("bytewire: connection lost, starting reconnect")
	reconnect()
}

// reconnect attempts to re-establish the connection with exponential backoff.
func reconnect() {
	// Clear stale DOM state — server will re-mount the full tree.
	clearNodeRegistry()
	root.Set("textContent", "")
	showOverlay("Reconnecting…")

	config := js.Global().Get("__bw_config")
	wtURL := config.Get("url").String()
	certHash := config.Get("certHash")

	delay := 1000 // milliseconds
	maxDelay := 10000
	maxTotalMs := 30000
	elapsedMs := 0

	for elapsedMs < maxTotalMs {
		// Sleep using a JS setTimeout-based channel.
		sleep(delay)
		elapsedMs += delay

		fmt.Printf("bytewire: reconnect attempt (elapsed %ds)\n", elapsedMs/1000)

		wt, err := dialWebTransport(wtURL, certHash)
		if err != "" {
			fmt.Println("bytewire: reconnect failed:", err)
			delay = min(delay*2, maxDelay)
			continue
		}

		if !openIntentStream(wt) {
			fmt.Println("bytewire: reconnect intent stream failed")
			delay = min(delay*2, maxDelay)
			continue
		}

		// Success
		fmt.Println("bytewire: reconnected")
		hideOverlay()
		go readIncomingStreamsWithReconnect(wt)
		return
	}

	// Exhausted retries
	showOverlay("Connection lost. Please reload the page.")
	fmt.Println("bytewire: reconnect failed after 30s")
}

// clearNodeRegistry removes all entries from the nodes map.
func clearNodeRegistry() {
	for id := range nodes {
		delete(nodes, id)
	}
}

// showOverlay displays a full-screen reconnection status overlay.
func showOverlay(msg string) {
	if !overlay.IsUndefined() && !overlay.IsNull() {
		overlay.Set("textContent", msg)
		return
	}
	overlay = document.Call("createElement", "div")
	overlay.Set("id", "bw-reconnect-overlay")
	overlay.Get("style").Set("cssText",
		"position:fixed;top:0;left:0;width:100%;height:100%;"+
			"background:rgba(0,0,0,0.7);color:#fff;"+
			"display:flex;align-items:center;justify-content:center;"+
			"font-family:system-ui,sans-serif;font-size:1.25rem;z-index:99999")
	overlay.Set("textContent", msg)
	document.Get("body").Call("appendChild", overlay)
}

// hideOverlay removes the reconnection overlay from the DOM.
func hideOverlay() {
	if overlay.IsUndefined() || overlay.IsNull() {
		return
	}
	parent := overlay.Get("parentNode")
	if !parent.IsNull() && !parent.IsUndefined() {
		parent.Call("removeChild", overlay)
	}
	overlay = js.Undefined()
}

// sleep blocks for the given number of milliseconds using setTimeout.
func sleep(ms int) {
	ch := make(chan struct{})
	js.Global().Call("setTimeout", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		close(ch)
		return nil
	}), ms)
	<-ch
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
				go readStreamAndPatch(stream, readNext)
				return nil
			}),
		).Call("catch",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				fmt.Println("bytewire: stream reader error:", args[0].Call("toString").String())
				return nil
			}),
		)
		return nil
	})

	readNext.Invoke()
}

// readStreamAndPatch fully reads a unidirectional stream and applies the opcodes.
func readStreamAndPatch(stream js.Value, readNext js.Func) {
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
				fmt.Println("bytewire: chunk read error:", args[0].Call("toString").String())
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
			fmt.Println("bytewire: send intent failed:", args[0].Call("toString").String())
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
			fmt.Println("bytewire: send nav failed:", args[0].Call("toString").String())
			return nil
		}),
	)
}

// findBWNode walks up from el to find the nearest element with data-bw-id.
func findBWNode(el js.Value) (uint32, bool) {
	body := document.Get("body")
	for !el.IsNull() && !el.IsUndefined() && !el.Equal(body) {
		attr := el.Call("getAttribute", "data-bw-id")
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

// setupEventDelegation attaches event listeners on #bw-root that
// map DOM events to Bytewire binary intents.
func setupEventDelegation() {
	// Click
	root.Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")
		if nodeID, ok := findBWNode(target); ok {
			e.Call("stopPropagation")
			sendIntent(nodeID, 0x01, nil) // EventClick
		}
		return nil
	}))

	// Input
	root.Call("addEventListener", "input", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")
		if nodeID, ok := findBWNode(target); ok {
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
		if nodeID, ok := findBWNode(target); ok {
			sendIntent(nodeID, 0x03, nil) // EventSubmit
		}
		return nil
	}))

	// SPA link interception: any <a> with href starting with "/"
	document.Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		e := args[0]
		target := e.Get("target")

		// Walk up to find nearest <a> with a local href
		body := document.Get("body")
		el := target
		for !el.IsNull() && !el.IsUndefined() && !el.Equal(body) {
			tagName := el.Get("tagName")
			if !tagName.IsUndefined() && tagName.String() == "A" {
				href := el.Call("getAttribute", "href")
				if !href.IsNull() && !href.IsUndefined() {
					h := href.String()
					if len(h) > 0 && h[0] == '/' {
						e.Call("preventDefault")
						e.Call("stopPropagation")
						sendClientNav(h)
						return nil
					}
				}
			}
			el = el.Get("parentElement")
		}
		return nil
	}))
}
