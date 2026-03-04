package protocol

import (
	"encoding/binary"
	"testing"
)

func TestEncodeHello(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeHello(1, 0)

	data := buf.Bytes()
	// Expected: [4B len=3][0x00][1][0]
	if len(data) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(data))
	}
	frameLen := binary.BigEndian.Uint32(data[0:4])
	if frameLen != 3 {
		t.Fatalf("expected frame length 3, got %d", frameLen)
	}
	if data[4] != OpHello {
		t.Fatalf("expected opcode 0x%02x, got 0x%02x", OpHello, data[4])
	}
	if data[5] != 1 {
		t.Fatalf("expected major 1, got %d", data[5])
	}
	if data[6] != 0 {
		t.Fatalf("expected minor 0, got %d", data[6])
	}
}

func TestEncodeClientHello(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeClientHello(1, 0)

	data := buf.Bytes()
	// Expected: [4B len=3][0x12][1][0]
	if len(data) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(data))
	}
	frameLen := binary.BigEndian.Uint32(data[0:4])
	if frameLen != 3 {
		t.Fatalf("expected frame length 3, got %d", frameLen)
	}
	if data[4] != OpClientHello {
		t.Fatalf("expected opcode 0x%02x, got 0x%02x", OpClientHello, data[4])
	}
	if data[5] != 1 {
		t.Fatalf("expected major 1, got %d", data[5])
	}
	if data[6] != 0 {
		t.Fatalf("expected minor 0, got %d", data[6])
	}
}

func TestDecodeHelloRoundTrip(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeHello(1, 2)

	msg, total, err := DecodeFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if total != 7 {
		t.Fatalf("expected 7 bytes consumed, got %d", total)
	}
	if msg.Op != OpHello {
		t.Fatalf("expected OpHello, got 0x%02x", msg.Op)
	}
	if msg.Major != 1 || msg.Minor != 2 {
		t.Fatalf("expected v1.2, got v%d.%d", msg.Major, msg.Minor)
	}
}

func TestDecodeClientHelloRoundTrip(t *testing.T) {
	buf := AcquireBuffer()
	defer buf.Release()

	buf.EncodeClientHello(1, 0)

	msg, total, err := DecodeFrame(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if total != 7 {
		t.Fatalf("expected 7 bytes consumed, got %d", total)
	}
	if msg.Op != OpClientHello {
		t.Fatalf("expected OpClientHello, got 0x%02x", msg.Op)
	}
	if msg.Major != 1 || msg.Minor != 0 {
		t.Fatalf("expected v1.0, got v%d.%d", msg.Major, msg.Minor)
	}
}
