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

// beginFrame reserves 4 bytes for the frame length prefix and returns the
// offset where the length will be written.
func (b *Buffer) beginFrame() int {
	off := len(b.buf)
	b.buf = append(b.buf, 0, 0, 0, 0) // placeholder for uint32 length
	return off
}

// endFrame patches the 4-byte length prefix at the given offset with the
// actual frame size (everything after the 4-byte prefix).
func (b *Buffer) endFrame(off int) {
	frameLen := uint32(len(b.buf) - off - 4)
	binary.BigEndian.PutUint32(b.buf[off:off+4], frameLen)
}

// EncodeUpdateText writes an OpUpdateText instruction.
func (b *Buffer) EncodeUpdateText(nodeID uint32, text string) {
	off := b.beginFrame()
	b.writeByte(OpUpdateText)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(text))
	b.endFrame(off)
}

// EncodeSetAttr writes an OpSetAttr instruction.
func (b *Buffer) EncodeSetAttr(nodeID uint32, key, value string) {
	off := b.beginFrame()
	b.writeByte(OpSetAttr)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(key))
	b.writeByte(0x00) // null separator
	b.writeBytes([]byte(value))
	b.endFrame(off)
}

// EncodeRemoveAttr writes an OpRemoveAttr instruction.
func (b *Buffer) EncodeRemoveAttr(nodeID uint32, key string) {
	off := b.beginFrame()
	b.writeByte(OpRemoveAttr)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(key))
	b.endFrame(off)
}

// EncodeInsertNode writes an OpInsertNode instruction.
func (b *Buffer) EncodeInsertNode(nodeID, parentID, siblingID uint32, tag string, attrs map[string]string) {
	off := b.beginFrame()
	b.writeByte(OpInsertNode)
	b.writeUint32(nodeID)
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
	b.endFrame(off)
}

// EncodeRemoveNode writes an OpRemoveNode instruction.
func (b *Buffer) EncodeRemoveNode(nodeID uint32) {
	off := b.beginFrame()
	b.writeByte(OpRemoveNode)
	b.writeUint32(nodeID)
	b.endFrame(off)
}

// EncodeSetStyle writes an OpSetStyle instruction.
func (b *Buffer) EncodeSetStyle(nodeID uint32, property, value string) {
	off := b.beginFrame()
	b.writeByte(OpSetStyle)
	b.writeUint32(nodeID)
	b.writeBytes([]byte(property))
	b.writeByte(0x00)
	b.writeBytes([]byte(value))
	b.endFrame(off)
}

// EncodePushHistory writes an OpPushHistory instruction.
func (b *Buffer) EncodePushHistory(path string) {
	off := b.beginFrame()
	b.writeByte(OpPushHistory)
	b.writeBytes([]byte(path))
	b.endFrame(off)
}

// EncodeClientIntent writes an OpClientIntent instruction.
func (b *Buffer) EncodeClientIntent(nodeID uint32, eventType byte, payload []byte) {
	off := b.beginFrame()
	b.writeByte(OpClientIntent)
	b.writeUint32(nodeID)
	b.writeByte(eventType)
	b.writeBytes(payload)
	b.endFrame(off)
}

// EncodeClientNav writes an OpClientNav instruction.
func (b *Buffer) EncodeClientNav(path string) {
	off := b.beginFrame()
	b.writeByte(OpClientNav)
	b.writeBytes([]byte(path))
	b.endFrame(off)
}
