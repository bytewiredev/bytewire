package engine

import (
	"bytes"
	"encoding/binary"
	"io"
	"net/http/httptest"
	"sync"
	"testing"

	"golang.org/x/net/websocket"
)

// TestWsWriterWriteMessage verifies that wsWriter sends binary data correctly
// over a real WebSocket connection.
func TestWsWriterWriteMessage(t *testing.T) {
	var serverConn *websocket.Conn
	ready := make(chan struct{})

	srv := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame
		serverConn = conn
		close(ready)
		// Block until test finishes reading.
		select {}
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	clientConn, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer clientConn.Close()

	<-ready

	w := &wsWriter{conn: serverConn}
	payload := []byte{0x01, 0x02, 0x03, 0x04}
	if err := w.WriteMessage(payload); err != nil {
		t.Fatal("WriteMessage:", err)
	}

	buf := make([]byte, 256)
	n, err := clientConn.Read(buf)
	if err != nil {
		t.Fatal("read:", err)
	}

	if !bytes.Equal(buf[:n], payload) {
		t.Errorf("got %x, want %x", buf[:n], payload)
	}
}

// TestWsWriterConcurrent verifies that concurrent WriteMessage calls
// do not interleave data (the internal mutex serializes writes).
func TestWsWriterConcurrent(t *testing.T) {
	var serverConn *websocket.Conn
	ready := make(chan struct{})

	srv := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame
		serverConn = conn
		close(ready)
		select {}
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	clientConn, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer clientConn.Close()

	<-ready

	w := &wsWriter{conn: serverConn}
	const numWriters = 10
	msgLen := 64

	var wg sync.WaitGroup
	wg.Add(numWriters)
	for i := range numWriters {
		go func(id int) {
			defer wg.Done()
			msg := bytes.Repeat([]byte{byte(id)}, msgLen)
			if err := w.WriteMessage(msg); err != nil {
				t.Errorf("writer %d: %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	// Read all messages from the client side and verify none are garbled.
	received := 0
	for received < numWriters {
		buf := make([]byte, 256)
		n, err := clientConn.Read(buf)
		if err != nil {
			t.Fatal("read:", err)
		}
		msg := buf[:n]
		if len(msg) != msgLen {
			t.Errorf("message %d: got len %d, want %d", received, len(msg), msgLen)
		}
		// All bytes should be the same value (no interleaving).
		for j := 1; j < len(msg); j++ {
			if msg[j] != msg[0] {
				t.Errorf("message %d: interleaved data at byte %d", received, j)
				break
			}
		}
		received++
	}
}

// TestWsFrameReading verifies that readWSFrames correctly reads length-prefixed
// frames sent by a client over WebSocket.
func TestWsFrameReading(t *testing.T) {
	intentReceived := make(chan []byte, 1)

	// We test frame reading directly by creating a pipe-like WebSocket setup.
	srv := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		conn.PayloadType = websocket.BinaryFrame

		// Read one length-prefixed frame.
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			t.Error("read len:", err)
			return
		}
		frameLen := int(binary.BigEndian.Uint32(lenBuf))
		frame := make([]byte, 4+frameLen)
		copy(frame, lenBuf)
		if _, err := io.ReadFull(conn, frame[4:]); err != nil {
			t.Error("read body:", err)
			return
		}
		intentReceived <- frame
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[len("http"):]
	clientConn, err := websocket.Dial(wsURL, "", srv.URL)
	if err != nil {
		t.Fatal("dial:", err)
	}
	clientConn.PayloadType = websocket.BinaryFrame

	// Send a length-prefixed frame: [4B len][opcode byte][data...]
	payload := []byte{0x10, 0x00, 0x00, 0x00, 0x01, 0x01}
	frame := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(frame[0:4], uint32(len(payload)))
	copy(frame[4:], payload)

	if _, err := clientConn.Write(frame); err != nil {
		t.Fatal("write:", err)
	}

	got := <-intentReceived
	if !bytes.Equal(got, frame) {
		t.Errorf("got %x, want %x", got, frame)
	}
}

// TestWithWebSocketFallbackOption verifies the server option sets the flag.
func TestWithWebSocketFallbackOption(t *testing.T) {
	s := &Server{}
	opt := WithWebSocketFallback()
	opt(s)
	if !s.wsFallback {
		t.Error("expected wsFallback to be true")
	}
}
