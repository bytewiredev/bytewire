package protocol

import (
	"encoding/binary"
	"io"
	"sync"
)

// Buffer is a reusable, zero-allocation binary encoder for CBS opcodes.
// It pools internal byte slices to minimize GC pressure on hot paths.
type Buffer struct {
	buf []byte
}

var bufferPool = sync.Pool{
	New: func() any {
		return &Buffer{buf: make([]byte, 0, 256)}
	},
}

// AcquireBuffer returns a pooled Buffer ready for writing.
func AcquireBuffer() *Buffer {
	b := bufferPool.Get().(*Buffer)
	b.buf = b.buf[:0]
	return b
}

// Release returns the Buffer to the pool.
func (b *Buffer) Release() {
	bufferPool.Put(b)
}

// Bytes returns the raw encoded bytes.
func (b *Buffer) Bytes() []byte {
	return b.buf
}

// Len returns the current buffer length.
func (b *Buffer) Len() int {
	return len(b.buf)
}

// WriteTo implements io.WriterTo for streaming directly to a connection.
func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.buf)
	return int64(n), err
}

// Reset clears the buffer for reuse without releasing to the pool.
func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
}

func (b *Buffer) writeByte(v byte) {
	b.buf = append(b.buf, v)
}

func (b *Buffer) writeUint32(v uint32) {
	b.buf = binary.BigEndian.AppendUint32(b.buf, v)
}

func (b *Buffer) writeBytes(data []byte) {
	b.buf = append(b.buf, data...)
}

// EncodeUpdateText writes an OpUpdateText instruction.
func (b *Buffer) EncodeUpdateText(nodeID uint32, text string) {
	b.writeByte(OpUpdateText)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(text))
}

// EncodeSetAttr writes an OpSetAttr instruction.
func (b *Buffer) EncodeSetAttr(nodeID uint32, key, value string) {
	b.writeByte(OpSetAttr)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(key))
	b.writeByte(0x00) // null separator
	b.writeBytes([]byte(value))
}

// EncodeRemoveAttr writes an OpRemoveAttr instruction.
func (b *Buffer) EncodeRemoveAttr(nodeID uint32, key string) {
	b.writeByte(OpRemoveAttr)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(key))
}

// EncodeInsertNode writes an OpInsertNode instruction.
func (b *Buffer) EncodeInsertNode(parentID, siblingID uint32, tag string, attrs map[string]string) {
	b.writeByte(OpInsertNode)
	b.writeUint32(parentID)
	b.writeUint32(siblingID)

	// Tag
	tagBytes := []byte(tag)
	b.writeByte(byte(len(tagBytes)))
	b.writeBytes(tagBytes)

	// Attributes
	attrCount := uint16(len(attrs))
	b.buf = binary.BigEndian.AppendUint16(b.buf, attrCount)
	for k, v := range attrs {
		kb := []byte(k)
		vb := []byte(v)
		b.buf = binary.BigEndian.AppendUint16(b.buf, uint16(len(kb)))
		b.writeBytes(kb)
		b.buf = binary.BigEndian.AppendUint16(b.buf, uint16(len(vb)))
		b.writeBytes(vb)
	}
}

// EncodeRemoveNode writes an OpRemoveNode instruction.
func (b *Buffer) EncodeRemoveNode(nodeID uint32) {
	b.writeByte(OpRemoveNode)
	b.writeUint32(nodeID)
}

// EncodeSetStyle writes an OpSetStyle instruction.
func (b *Buffer) EncodeSetStyle(nodeID uint32, property, value string) {
	b.writeByte(OpSetStyle)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(property))
	b.writeByte(0x00)
	b.writeBytes([]byte(value))
}

// EncodePushHistory writes an OpPushHistory instruction.
func (b *Buffer) EncodePushHistory(path string) {
	b.writeByte(OpPushHistory)
	b.writeBytes([]byte(path))
}

// EncodeClientIntent writes an OpClientIntent instruction.
func (b *Buffer) EncodeClientIntent(nodeID uint32, eventType byte, payload []byte) {
	b.writeByte(OpClientIntent)
	b.writeUint32(nodeID)
	b.writeByte(eventType)
	b.writeBytes(payload)
}

// EncodeClientNav writes an OpClientNav instruction.
func (b *Buffer) EncodeClientNav(path string) {
	b.writeByte(OpClientNav)
	b.writeBytes([]byte(path))
}
