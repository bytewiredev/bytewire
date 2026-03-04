package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrShortRead    = errors.New("cbs: unexpected end of message")
	ErrUnknownOp    = errors.New("cbs: unknown opcode")
	ErrInvalidFrame = errors.New("cbs: invalid frame structure")
)

// Message is a decoded CBS binary instruction.
type Message struct {
	Op        byte
	NodeID    uint32
	ParentID  uint32   // OpInsertNode
	SiblingID uint32   // OpInsertNode
	Tag       string   // OpInsertNode
	Attrs     [][2]string // key-value pairs
	Key       string   // OpSetAttr, OpRemoveAttr, OpSetStyle
	Value     string   // OpSetAttr, OpSetStyle
	Text      string   // OpUpdateText, OpPushHistory, OpClientNav
	EventType byte     // OpClientIntent
	Payload   []byte   // OpClientIntent
}

// Decode reads a single CBS message from raw bytes and returns
// the decoded Message plus the number of bytes consumed.
func Decode(data []byte) (Message, int, error) {
	if len(data) < 1 {
		return Message{}, 0, ErrShortRead
	}

	var msg Message
	msg.Op = data[0]
	pos := 1

	switch msg.Op {
	case OpUpdateText:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		msg.Text = string(data[5:])
		return msg, len(data), nil

	case OpSetAttr:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		rest := data[5:]
		sep := findNull(rest)
		if sep < 0 {
			return msg, 0, ErrInvalidFrame
		}
		msg.Key = string(rest[:sep])
		msg.Value = string(rest[sep+1:])
		return msg, len(data), nil

	case OpRemoveAttr:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		msg.Key = string(data[5:])
		return msg, len(data), nil

	case OpInsertNode:
		if len(data) < 10 {
			return msg, 0, ErrShortRead
		}
		msg.ParentID = binary.BigEndian.Uint32(data[1:5])
		msg.SiblingID = binary.BigEndian.Uint32(data[5:9])
		pos = 9

		// Tag
		if pos >= len(data) {
			return msg, 0, ErrShortRead
		}
		tagLen := int(data[pos])
		pos++
		if pos+tagLen > len(data) {
			return msg, 0, ErrShortRead
		}
		msg.Tag = string(data[pos : pos+tagLen])
		pos += tagLen

		// Attrs
		if pos+2 > len(data) {
			return msg, 0, ErrShortRead
		}
		attrCount := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		msg.Attrs = make([][2]string, 0, attrCount)
		for i := 0; i < attrCount; i++ {
			if pos+2 > len(data) {
				return msg, 0, ErrShortRead
			}
			kLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			pos += 2
			if pos+kLen > len(data) {
				return msg, 0, ErrShortRead
			}
			k := string(data[pos : pos+kLen])
			pos += kLen

			if pos+2 > len(data) {
				return msg, 0, ErrShortRead
			}
			vLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			pos += 2
			if pos+vLen > len(data) {
				return msg, 0, ErrShortRead
			}
			v := string(data[pos : pos+vLen])
			pos += vLen

			msg.Attrs = append(msg.Attrs, [2]string{k, v})
		}
		return msg, pos, nil

	case OpRemoveNode:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		return msg, 5, nil

	case OpSetStyle:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		rest := data[5:]
		sep := findNull(rest)
		if sep < 0 {
			return msg, 0, ErrInvalidFrame
		}
		msg.Key = string(rest[:sep])
		msg.Value = string(rest[sep+1:])
		return msg, len(data), nil

	case OpPushHistory:
		msg.Text = string(data[1:])
		return msg, len(data), nil

	case OpClientIntent:
		if len(data) < 6 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		msg.EventType = data[5]
		msg.Payload = data[6:]
		return msg, len(data), nil

	case OpClientNav:
		msg.Text = string(data[1:])
		return msg, len(data), nil

	default:
		return msg, 0, fmt.Errorf("%w: 0x%02x", ErrUnknownOp, msg.Op)
	}
}

func findNull(data []byte) int {
	for i, b := range data {
		if b == 0x00 {
			return i
		}
	}
	return -1
}
