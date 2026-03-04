package protocol

import (
	"testing"
)

func TestEncodeDecodeUpdateText(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeUpdateText(1024, "Hello CBS")
	msg, n, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if n != buf.Len() {
		t.Fatalf("expected %d bytes consumed, got %d", buf.Len(), n)
	}
	if msg.Op != OpUpdateText {
		t.Fatalf("expected op 0x%02x, got 0x%02x", OpUpdateText, msg.Op)
	}
	if msg.NodeID != 1024 {
		t.Fatalf("expected nodeID 1024, got %d", msg.NodeID)
	}
	if msg.Text != "Hello CBS" {
		t.Fatalf("expected text %q, got %q", "Hello CBS", msg.Text)
	}
}

func TestEncodeDecodeSetAttr(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeSetAttr(42, "class", "active")
	msg, _, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if msg.Op != OpSetAttr {
		t.Fatalf("expected OpSetAttr, got 0x%02x", msg.Op)
	}
	if msg.NodeID != 42 {
		t.Fatalf("expected nodeID 42, got %d", msg.NodeID)
	}
	if msg.Key != "class" || msg.Value != "active" {
		t.Fatalf("expected class=active, got %s=%s", msg.Key, msg.Value)
	}
}

func TestEncodeDecodeRemoveNode(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeRemoveNode(99)
	msg, n, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5 bytes, got %d", n)
	}
	if msg.Op != OpRemoveNode || msg.NodeID != 99 {
		t.Fatalf("unexpected: op=0x%02x node=%d", msg.Op, msg.NodeID)
	}
}

func TestEncodeDecodeInsertNode(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	attrs := map[string]string{"id": "btn1", "class": "primary"}
	buf.EncodeInsertNode(1, 0, "button", attrs)

	msg, _, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if msg.Op != OpInsertNode {
		t.Fatalf("expected OpInsertNode, got 0x%02x", msg.Op)
	}
	if msg.ParentID != 1 || msg.SiblingID != 0 {
		t.Fatalf("expected parent=1 sibling=0, got %d %d", msg.ParentID, msg.SiblingID)
	}
	if msg.Tag != "button" {
		t.Fatalf("expected tag 'button', got %q", msg.Tag)
	}
	if len(msg.Attrs) != 2 {
		t.Fatalf("expected 2 attrs, got %d", len(msg.Attrs))
	}
}

func TestEncodeDecodePushHistory(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodePushHistory("/dashboard/settings")
	msg, _, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if msg.Op != OpPushHistory || msg.Text != "/dashboard/settings" {
		t.Fatalf("unexpected: op=0x%02x path=%q", msg.Op, msg.Text)
	}
}

func TestEncodeDecodeClientIntent(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeClientIntent(512, EventClick, []byte("x=10,y=20"))
	msg, _, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if msg.Op != OpClientIntent || msg.NodeID != 512 || msg.EventType != EventClick {
		t.Fatalf("unexpected: op=0x%02x node=%d event=0x%02x", msg.Op, msg.NodeID, msg.EventType)
	}
	if string(msg.Payload) != "x=10,y=20" {
		t.Fatalf("expected payload %q, got %q", "x=10,y=20", string(msg.Payload))
	}
}

func BenchmarkEncodeUpdateText(b *testing.B) {
	buf := AcquireBuffer()
	defer buf.Release()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.EncodeUpdateText(1024, "Hello CBS")
	}
}

func BenchmarkDecodeUpdateText(b *testing.B) {
	buf := AcquireBuffer()
	buf.EncodeUpdateText(1024, "Hello CBS")
	data := buf.Bytes()
	buf.Release()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = Decode(data)
	}
}
