//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

// applyOpcodes processes a binary opcode stream and applies DOM mutations.
// Each opcode is wrapped in a 4-byte length-prefixed frame.
func applyOpcodes(data []byte) {
	pos := 0
	for pos < len(data) {
		// Read frame length prefix
		if pos+4 > len(data) {
			return
		}
		frameLen := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if pos+frameLen > len(data) {
			return
		}
		frame := data[pos : pos+frameLen]
		pos += frameLen

		applyFrame(frame)
	}
}

// applyFrame processes a single bounded opcode frame.
func applyFrame(data []byte) {
	if len(data) < 1 {
		return
	}
	op := data[0]
	p := 1 // position within this frame

	switch op {
	case 0x01: // OpUpdateText
		if p+4 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		text := string(data[p:])

		if node, ok := nodes[nodeID]; ok {
			node.Set("textContent", text)
		}

	case 0x02: // OpSetAttr
		if p+4 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		rest := data[p:]
		sep := findNull(rest)
		if sep < 0 {
			return
		}
		key := string(rest[:sep])
		value := string(rest[sep+1:])

		if node, ok := nodes[nodeID]; ok {
			node.Call("setAttribute", key, value)
		}

	case 0x03: // OpRemoveAttr
		if p+4 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		key := string(data[p:])

		if node, ok := nodes[nodeID]; ok {
			node.Call("removeAttribute", key)
		}

	case 0x04: // OpInsertNode
		if p+12 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		parentID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		_ = binary.BigEndian.Uint32(data[p : p+4]) // siblingID (TODO)
		p += 4

		if p >= len(data) {
			return
		}
		tagLen := int(data[p])
		p++
		if p+tagLen > len(data) {
			return
		}
		tag := string(data[p : p+tagLen])
		p += tagLen

		if p+2 > len(data) {
			return
		}
		attrCount := int(binary.BigEndian.Uint16(data[p : p+2]))
		p += 2

		var el js.Value
		if tag == "#text" {
			el = document.Call("createTextNode", "")
		} else {
			el = document.Call("createElement", tag)
		}

		for range attrCount {
			if p+2 > len(data) {
				return
			}
			kLen := int(binary.BigEndian.Uint16(data[p : p+2]))
			p += 2
			if p+kLen > len(data) {
				return
			}
			k := string(data[p : p+kLen])
			p += kLen

			if p+2 > len(data) {
				return
			}
			vLen := int(binary.BigEndian.Uint16(data[p : p+2]))
			p += 2
			if p+vLen > len(data) {
				return
			}
			v := string(data[p : p+vLen])
			p += vLen

			el.Call("setAttribute", k, v)
		}

		nodes[nodeID] = el

		if parent, ok := nodes[parentID]; ok {
			parent.Call("appendChild", el)
		} else if parentID == 0 {
			root.Call("appendChild", el)
		}

	case 0x05: // OpRemoveNode
		if p+4 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])

		if node, ok := nodes[nodeID]; ok {
			parent := node.Get("parentNode")
			if !parent.IsNull() {
				parent.Call("removeChild", node)
			}
			delete(nodes, nodeID)
		}

	case 0x07: // OpSetStyle
		if p+4 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		rest := data[p:]
		sep := findNull(rest)
		if sep < 0 {
			return
		}
		prop := string(rest[:sep])
		val := string(rest[sep+1:])

		if node, ok := nodes[nodeID]; ok {
			node.Get("style").Call("setProperty", prop, val)
		}

	case 0x08: // OpPushHistory
		path := string(data[p:])
		js.Global().Get("history").Call("pushState", nil, "", path)

	default:
		fmt.Printf("cbs: unknown opcode 0x%02x\n", op)
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
