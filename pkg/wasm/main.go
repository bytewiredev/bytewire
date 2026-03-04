//go:build js && wasm

// Package wasm is the Bytewire browser client. It connects via WebTransport
// (or falls back to WebSocket), receives binary opcodes, patches the DOM,
// and delegates events -- all from Go via syscall/js. No external JavaScript required.
package wasm

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"sync/atomic"
	"syscall/js"
)

var (
	document js.Value
	root     js.Value
	nodes    map[uint32]js.Value

	// conn is the active transport connection (WebTransport or WebSocket).
	conn transport

	// overlay is the reconnection status overlay element.
	overlay js.Value
)

func init() {
	document = js.Global().Get("document")
	nodes = make(map[uint32]js.Value)
	overlay = js.Undefined()
}

// Start initializes the Bytewire WASM client: connects to the server,
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

// connect establishes a transport connection and starts reading messages.
// Returns true on success, false on fatal config errors.
func connect() bool {
	config := js.Global().Get("__bw_config")
	if config.IsUndefined() || config.IsNull() {
		root.Set("textContent", "Error: __bw_config not set")
		return false
	}

	var err error

	if detectTransport() {
		wtURL := config.Get("url").String()
		certHash := config.Get("certHash")
		conn, err = newWebTransportConn(wtURL, certHash)
		if err != nil {
			fmt.Println("bytewire: WebTransport failed:", err)
			// Fall through to WebSocket if available
			conn, err = tryWebSocket(config)
			if err != nil {
				root.Set("textContent", "Connection failed: "+err.Error())
				return false
			}
		} else {
			fmt.Println("bytewire: WebTransport connected")
		}
	} else {
		fmt.Println("bytewire: WebTransport not available, using WebSocket fallback")
		conn, err = tryWebSocket(config)
		if err != nil {
			root.Set("textContent", "Connection failed: "+err.Error())
			return false
		}
	}

	root.Set("textContent", "")
	atomic.StoreInt32(&IsOnline, 1)

	// Read incoming messages and apply DOM patches.
	conn.onMessage(func(data []byte) {
		applyOpcodes(data)
	})

	// Reconnect on close.
	conn.onClose(func() {
		atomic.StoreInt32(&IsOnline, 0)
		fmt.Println("bytewire: connection lost, starting reconnect")
		reconnect()
	})

	return true
}

// tryWebSocket attempts a WebSocket connection using the config's wsURL.
func tryWebSocket(config js.Value) (*webSocketConn, error) {
	wsURLVal := config.Get("wsURL")
	if wsURLVal.IsUndefined() || wsURLVal.IsNull() {
		return nil, fmt.Errorf("no WebSocket fallback URL configured")
	}
	wsConn, err := newWebSocketConn(wsURLVal.String())
	if err != nil {
		return nil, fmt.Errorf("WebSocket: %w", err)
	}
	fmt.Println("bytewire: WebSocket connected")
	return wsConn, nil
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

// reconnect attempts to re-establish the connection with exponential backoff.
func reconnect() {
	clearNodeRegistry()
	root.Set("textContent", "")
	showOverlay("Reconnecting…")
	loadPersistedQueue()

	delay := 1000
	maxDelay := 10000
	maxTotalMs := 30000
	elapsedMs := 0

	config := js.Global().Get("__bw_config")

	for elapsedMs < maxTotalMs {
		sleep(delay)
		elapsedMs += delay

		fmt.Printf("bytewire: reconnect attempt (elapsed %ds)\n", elapsedMs/1000)

		var newConn transport
		var err error

		if detectTransport() {
			wtURL := config.Get("url").String()
			certHash := config.Get("certHash")
			newConn, err = newWebTransportConn(wtURL, certHash)
		}
		if newConn == nil || err != nil {
			wsConn, wsErr := tryWebSocket(config)
			if wsErr != nil {
				fmt.Println("bytewire: reconnect failed:", wsErr)
				delay = min(delay*2, maxDelay)
				continue
			}
			newConn = wsConn
		}

		conn = newConn
		fmt.Println("bytewire: reconnected")
		hideOverlay()
		drainOfflineQueue()
		clearPersistedQueue()

		conn.onMessage(func(data []byte) {
			applyOpcodes(data)
		})
		conn.onClose(func() {
			fmt.Println("bytewire: connection lost, starting reconnect")
			reconnect()
		})
		return
	}

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

// sendIntent encodes and sends an OpClientIntent frame over the active transport.
func sendIntent(nodeID uint32, eventType byte, payload []byte) {
	frameLen := 1 + 4 + 1 + len(payload)
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x10 // OpClientIntent
	binary.BigEndian.PutUint32(frame[5:9], nodeID)
	frame[9] = eventType
	copy(frame[10:], payload)

	if atomic.LoadInt32(&IsOnline) != 1 {
		GlobalQueue.Enqueue(frame)
		persistQueue()
		return
	}

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)
	conn.send(uint8Array)
}

// sendClientNav encodes and sends an OpClientNav frame over the active transport.
func sendClientNav(path string) {
	frameLen := 1 + len(path)
	frame := make([]byte, 4+frameLen)
	binary.BigEndian.PutUint32(frame[0:4], uint32(frameLen))
	frame[4] = 0x11 // OpClientNav
	copy(frame[5:], path)

	if atomic.LoadInt32(&IsOnline) != 1 {
		GlobalQueue.Enqueue(frame)
		persistQueue()
		return
	}

	uint8Array := js.Global().Get("Uint8Array").New(len(frame))
	js.CopyBytesToJS(uint8Array, frame)
	conn.send(uint8Array)
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
