//go:build js && wasm

package wasm

import (
	"fmt"
	"syscall/js"
)

// transport abstracts the browser-side connection so the WASM client
// can use either WebTransport or WebSocket transparently.
type transport interface {
	// send writes binary data to the server.
	send(data js.Value)
	// onMessage registers a callback invoked with each incoming binary message.
	onMessage(callback func([]byte))
	// onClose registers a callback invoked when the connection closes.
	onClose(callback func())
	// close shuts down the connection.
	close()
}

// --- WebTransport adapter ---

type webTransportConn struct {
	wt           js.Value
	intentWriter js.Value
}

func newWebTransportConn(wtURL string, certHash js.Value) (*webTransportConn, error) {
	wt, errStr := dialWebTransport(wtURL, certHash)
	if errStr != "" {
		return nil, fmt.Errorf("%s", errStr)
	}

	// Open the persistent bidi stream for client intents.
	writer, ok := openBidiWriter(wt)
	if !ok {
		return nil, fmt.Errorf("failed to open intent stream")
	}

	return &webTransportConn{wt: wt, intentWriter: writer}, nil
}

func (c *webTransportConn) send(data js.Value) {
	c.intentWriter.Call("write", data).Call("catch",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			fmt.Println("bytewire: send intent failed:", args[0].Call("toString").String())
			return nil
		}),
	)
}

func (c *webTransportConn) onMessage(callback func([]byte)) {
	go readIncomingStreams(c.wt, callback)
}

func (c *webTransportConn) onClose(callback func()) {
	c.wt.Get("closed").Call("then",
		js.FuncOf(func(_ js.Value, _ []js.Value) any {
			callback()
			return nil
		}),
	).Call("catch",
		js.FuncOf(func(_ js.Value, _ []js.Value) any {
			callback()
			return nil
		}),
	)
}

func (c *webTransportConn) close() {
	c.wt.Call("close")
}

// openBidiWriter opens a bidi stream and returns the writable writer.
func openBidiWriter(wt js.Value) (js.Value, bool) {
	openDone := make(chan struct{})
	var writer js.Value
	wt.Call("createBidirectionalStream").Call("then",
		js.FuncOf(func(_ js.Value, args []js.Value) any {
			stream := args[0]
			writable := stream.Get("writable")
			writer = writable.Call("getWriter")
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
	return writer, !writer.IsUndefined() && !writer.IsNull()
}

// readIncomingStreams reads unidirectional streams from the WT session
// and invokes the callback with each complete message's bytes.
func readIncomingStreams(wt js.Value, callback func([]byte)) {
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
				go readStreamFull(stream, callback, readNext)
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

// readStreamFull reads all chunks from a unidirectional stream, concatenates them,
// and invokes the callback with the full byte payload.
func readStreamFull(stream js.Value, callback func([]byte), readNext js.Func) {
	sr := stream.Call("getReader")
	var chunks []js.Value

	var readChunk js.Func
	readChunk = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		sr.Call("read").Call("then",
			js.FuncOf(func(_ js.Value, args []js.Value) any {
				result := args[0]
				if result.Get("done").Bool() {
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
					callback(data)

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

// --- WebSocket adapter ---

type webSocketConn struct {
	ws js.Value
}

func newWebSocketConn(wsURL string) (*webSocketConn, error) {
	ws := js.Global().Get("WebSocket").New(wsURL)
	ws.Set("binaryType", "arraybuffer")

	done := make(chan error, 1)

	ws.Call("addEventListener", "open", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		done <- nil
		return nil
	}))
	ws.Call("addEventListener", "error", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		done <- fmt.Errorf("WebSocket connection failed")
		return nil
	}))

	if err := <-done; err != nil {
		return nil, err
	}

	return &webSocketConn{ws: ws}, nil
}

func (c *webSocketConn) send(data js.Value) {
	c.ws.Call("send", data.Get("buffer"))
}

func (c *webSocketConn) onMessage(callback func([]byte)) {
	c.ws.Call("addEventListener", "message", js.FuncOf(func(_ js.Value, args []js.Value) any {
		event := args[0]
		arrayBuf := event.Get("data")
		uint8arr := js.Global().Get("Uint8Array").New(arrayBuf)
		data := make([]byte, uint8arr.Get("byteLength").Int())
		js.CopyBytesToGo(data, uint8arr)
		callback(data)
		return nil
	}))
}

func (c *webSocketConn) onClose(callback func()) {
	c.ws.Call("addEventListener", "close", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		callback()
		return nil
	}))
}

func (c *webSocketConn) close() {
	c.ws.Call("close")
}

// detectTransport checks if the browser supports WebTransport.
func detectTransport() bool {
	wt := js.Global().Get("WebTransport")
	return !wt.IsUndefined() && !wt.IsNull()
}
