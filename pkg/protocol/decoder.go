package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrShortRead    = errors.New("bytewire: unexpected end of message")
	ErrUnknownOp    = errors.New("bytewire: unknown opcode")
	ErrInvalidFrame = errors.New("bytewire: invalid frame structure")
)

// Message is a decoded Bytewire binary instruction.
type Message struct {
	Op        byte
	Major     byte     // OpHello, OpClientHello
	Minor     byte     // OpHello, OpClientHello
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
	Offset    uint32   // OpReplaceText
	Length    uint32   // OpReplaceText
	Children  []Message // OpBatch
}

// Decode reads a single Bytewire message from raw bytes and returns
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
		if len(data) < 14 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		msg.ParentID = binary.BigEndian.Uint32(data[5:9])
		msg.SiblingID = binary.BigEndian.Uint32(data[9:13])
		pos = 13

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

	case OpReplaceText:
		if len(data) < 13 {
			return msg, 0, ErrShortRead
		}
		msg.NodeID = binary.BigEndian.Uint32(data[1:5])
		msg.Offset = binary.BigEndian.Uint32(data[5:9])
		msg.Length = binary.BigEndian.Uint32(data[9:13])
		msg.Text = string(data[13:])
		return msg, len(data), nil

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

	case OpBatch:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		count := binary.BigEndian.Uint32(data[1:5])
		children, err := decodeN(data[5:], count)
		if err != nil {
			return msg, 0, err
		}
		msg.Children = children
		return msg, len(data), nil

	case OpError:
		if len(data) < 3 {
			return msg, 0, ErrShortRead
		}
		msgLen := int(binary.BigEndian.Uint16(data[1:3]))
		if len(data) < 3+msgLen {
			return msg, 0, ErrShortRead
		}
		msg.Text = string(data[3 : 3+msgLen])
		return msg, 3 + msgLen, nil

	case OpDevToolsState:
		if len(data) < 5 {
			return msg, 0, ErrShortRead
		}
		jsonLen := int(binary.BigEndian.Uint32(data[1:5]))
		if len(data) < 5+jsonLen {
			return msg, 0, ErrShortRead
		}
		msg.Payload = data[5 : 5+jsonLen]
		return msg, 5 + jsonLen, nil

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

	case OpHello:
		if len(data) < 3 {
			return msg, 0, ErrShortRead
		}
		msg.Major = data[1]
		msg.Minor = data[2]
		return msg, 3, nil

	case OpClientHello:
		if len(data) < 3 {
			return msg, 0, ErrShortRead
		}
		msg.Major = data[1]
		msg.Minor = data[2]
		return msg, 3, nil

	default:
		return msg, 0, fmt.Errorf("%w: 0x%02x", ErrUnknownOp, msg.Op)
	}
}

// DecodeFrame reads a 4-byte length prefix, then decodes the opcode frame
// within that boundary. Returns the decoded message and total bytes consumed
// (including the 4-byte prefix).
func DecodeFrame(data []byte) (Message, int, error) {
	if len(data) < 4 {
		return Message{}, 0, ErrShortRead
	}
	frameLen := int(binary.BigEndian.Uint32(data[0:4]))
	if len(data) < 4+frameLen {
		return Message{}, 0, ErrShortRead
	}
	msg, _, err := Decode(data[4 : 4+frameLen])
	if err != nil {
		return msg, 0, err
	}
	return msg, 4 + frameLen, nil
}

// DecodeAll reads all length-prefixed frames from data and returns them.
func DecodeAll(data []byte) ([]Message, error) {
	var msgs []Message
	pos := 0
	for pos < len(data) {
		msg, n, err := DecodeFrame(data[pos:])
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, msg)
		pos += n
	}
	return msgs, nil
}

// decodeN decodes exactly n length-prefixed frames from data.
func decodeN(data []byte, n uint32) ([]Message, error) {
	msgs := make([]Message, 0, n)
	pos := 0
	for i := uint32(0); i < n; i++ {
		m, consumed, err := DecodeFrame(data[pos:])
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, m)
		pos += consumed
	}
	return msgs, nil
}

func findNull(data []byte) int {
	for i, b := range data {
		if b == 0x00 {
			return i
		}
	}
	return -1
}
