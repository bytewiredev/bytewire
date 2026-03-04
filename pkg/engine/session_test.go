package engine

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sync"
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/protocol"
)

// mockWriter captures binary messages written by the session.
type mockWriter struct {
	mu       sync.Mutex
	messages [][]byte
	closed   bool
}

func (w *mockWriter) WriteMessage(data []byte) error {
	w.mu.Lock()
	cp := make([]byte, len(data))
	copy(cp, data)
	w.messages = append(w.messages, cp)
	w.mu.Unlock()
	return nil
}

func (w *mockWriter) Close() error {
	w.mu.Lock()
	w.closed = true
	w.mu.Unlock()
	return nil
}

func (w *mockWriter) lastMessage() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.messages) == 0 {
		return nil
	}
	return w.messages[len(w.messages)-1]
}

func (w *mockWriter) messageCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.messages)
}

// newTestSession creates a session with a mock writer for testing.
func newTestSession() (*Session, *mockWriter) {
	w := &mockWriter{}
	logger := slog.Default()
	sess := NewSession(context.Background(), w, logger)
	return sess, w
}

// extractOpcodes parses length-prefixed frames from a binary message and
// returns the opcode byte from each frame.
func extractOpcodes(data []byte) []byte {
	var ops []byte
	pos := 0
	for pos < len(data) {
		if pos+4 > len(data) {
			break
		}
		frameLen := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		if pos+4+frameLen > len(data) {
			break
		}
		if frameLen > 0 {
			ops = append(ops, data[pos+4]) // first byte after length is the opcode
		}
		pos += 4 + frameLen
	}
	return ops
}

func TestFlush_NoChanges_NoWrite(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	count := dom.NewSignal(0)
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(dom.Children(
			dom.TextF(count, func(v int) string { return "hello" }),
		))
	})

	initialCount := w.messageCount()

	// Flush with no signal changes should not write anything.
	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	if w.messageCount() != initialCount {
		t.Errorf("expected no new messages, got %d", w.messageCount()-initialCount)
	}
}

func TestFlush_SingleSignalChange(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	count := dom.NewSignal(0)
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(dom.Children(
			dom.TextF(count, func(v int) string { return "v" }),
		))
	})

	initialCount := w.messageCount()

	// Change the signal and flush.
	count.Set(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	if w.messageCount() != initialCount+1 {
		t.Fatalf("expected 1 new message, got %d", w.messageCount()-initialCount)
	}

	// The message should contain an OpUpdateText.
	ops := extractOpcodes(w.lastMessage())
	if len(ops) == 0 {
		t.Fatal("expected at least one opcode in flush message")
	}
	found := false
	for _, op := range ops {
		if op == protocol.OpUpdateText {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected OpUpdateText in flush, got opcodes: %v", ops)
	}
}

func TestFlush_MultipleSignalChanges_Coalesced(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	count := dom.NewSignal(0)
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(dom.Children(
			dom.TextF(count, func(v int) string { return "v" }),
		))
	})

	initialCount := w.messageCount()

	// Multiple signal changes before flush should coalesce into one write.
	count.Set(1)
	count.Set(2)
	count.Set(3)

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	// Should produce exactly one new message (coalesced).
	newMessages := w.messageCount() - initialCount
	if newMessages != 1 {
		t.Errorf("expected 1 coalesced message, got %d", newMessages)
	}
}

func TestFlush_AfterFlush_Clean(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	count := dom.NewSignal(0)
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(dom.Children(
			dom.TextF(count, func(v int) string { return "v" }),
		))
	})

	// Change and flush.
	count.Set(1)
	sess.Flush()
	countAfterFirst := w.messageCount()

	// Second flush with no new changes should be a no-op.
	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}
	if w.messageCount() != countAfterFirst {
		t.Errorf("expected no new messages after clean flush, got %d", w.messageCount()-countAfterFirst)
	}
}

func TestHandleIntent_FlushesAfterHandler(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	count := dom.NewSignal(0)
	var buttonID dom.NodeID

	sess.Mount(func(s *Session) *dom.Node {
		btn := dom.Button(
			dom.OnClick(func(_ []byte) {
				count.Update(func(v int) int { return v + 1 })
			}),
			dom.Children(dom.Text("click")),
		)
		buttonID = btn.ID
		return dom.Div(dom.Children(
			dom.TextF(count, func(v int) string { return "v" }),
			btn,
		))
	})

	initialCount := w.messageCount()

	// Build a ClientIntent frame targeting the button.
	buf := protocol.AcquireBuffer()
	buf.EncodeClientIntent(uint32(buttonID), protocol.EventClick, nil)
	frame := buf.Bytes()
	buf.Release()

	if err := sess.HandleIntent(frame); err != nil {
		t.Fatalf("HandleIntent returned error: %v", err)
	}

	// Should have flushed the signal change automatically.
	if w.messageCount() <= initialCount {
		t.Error("expected HandleIntent to flush dirty nodes, but no new messages were sent")
	}
}

func TestFlush_AttrChange(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	enabled := dom.NewSignal(false)
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(
			dom.AttrF(enabled, "class", func(v bool) string {
				if v {
					return "active"
				}
				return "inactive"
			}),
		)
	})

	initialCount := w.messageCount()

	enabled.Set(true)
	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	if w.messageCount() != initialCount+1 {
		t.Fatalf("expected 1 new message for attr change, got %d", w.messageCount()-initialCount)
	}

	ops := extractOpcodes(w.lastMessage())
	found := false
	for _, op := range ops {
		if op == protocol.OpSetAttr {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected OpSetAttr in flush, got opcodes: %v", ops)
	}
}

func TestFlush_StyleChange(t *testing.T) {
	sess, w := newTestSession()
	defer sess.Close()

	color := dom.NewSignal("red")
	sess.Mount(func(s *Session) *dom.Node {
		return dom.Div(
			dom.StyleF(color, "color", func(v string) string { return v }),
		)
	})

	initialCount := w.messageCount()

	color.Set("blue")
	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	if w.messageCount() != initialCount+1 {
		t.Fatalf("expected 1 new message for style change, got %d", w.messageCount()-initialCount)
	}

	ops := extractOpcodes(w.lastMessage())
	found := false
	for _, op := range ops {
		if op == protocol.OpSetStyle {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected OpSetStyle in flush, got opcodes: %v", ops)
	}
}
