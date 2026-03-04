//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

// applyOpcodes processes a binary opcode stream and applies DOM mutations.
func applyOpcodes(data []byte) {
	pos := 0
	for pos < len(data) {
		if pos >= len(data) {
			break
		}
		op := data[pos]
		pos++

		switch op {
		case 0x01: // OpUpdateText
			if pos+4 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4
			// Read until end of frame (for single-message frames)
			// In batch mode, we'd need length-prefixed strings
			text := string(data[pos:])
			pos = len(data)

			if node, ok := nodes[nodeID]; ok {
				node.Set("textContent", text)
			}

		case 0x02: // OpSetAttr
			if pos+4 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4
			rest := data[pos:]
			sep := findNull(rest)
			if sep < 0 {
				return
			}
			key := string(rest[:sep])
			value := string(rest[sep+1:])
			pos = len(data)

			if node, ok := nodes[nodeID]; ok {
				node.Call("setAttribute", key, value)
			}

		case 0x03: // OpRemoveAttr
			if pos+4 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4
			key := string(data[pos:])
			pos = len(data)

			if node, ok := nodes[nodeID]; ok {
				node.Call("removeAttribute", key)
			}

		case 0x04: // OpInsertNode
			if pos+9 > len(data) {
				return
			}
			parentID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4
			_ = binary.BigEndian.Uint32(data[pos : pos+4]) // siblingID (TODO)
			pos += 4

			tagLen := int(data[pos])
			pos++
			if pos+tagLen > len(data) {
				return
			}
			tag := string(data[pos : pos+tagLen])
			pos += tagLen

			if pos+2 > len(data) {
				return
			}
			attrCount := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			pos += 2

			el := document.Call("createElement", tag)

			for range attrCount {
				if pos+2 > len(data) {
					return
				}
				kLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
				pos += 2
				if pos+kLen > len(data) {
					return
				}
				k := string(data[pos : pos+kLen])
				pos += kLen

				if pos+2 > len(data) {
					return
				}
				vLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
				pos += 2
				if pos+vLen > len(data) {
					return
				}
				v := string(data[pos : pos+vLen])
				pos += vLen

				el.Call("setAttribute", k, v)
			}

			// Assign a synthetic node ID from the parent+child count
			nodeID := uint32(len(nodes) + 1)
			nodes[nodeID] = el

			if parent, ok := nodes[parentID]; ok {
				parent.Call("appendChild", el)
			} else if parentID == 0 {
				root.Call("appendChild", el)
			}

		case 0x05: // OpRemoveNode
			if pos+4 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4

			if node, ok := nodes[nodeID]; ok {
				parent := node.Get("parentNode")
				if !parent.IsNull() {
					parent.Call("removeChild", node)
				}
				delete(nodes, nodeID)
			}

		case 0x07: // OpSetStyle
			if pos+4 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[pos : pos+4])
			pos += 4
			rest := data[pos:]
			sep := findNull(rest)
			if sep < 0 {
				return
			}
			prop := string(rest[:sep])
			val := string(rest[sep+1:])
			pos = len(data)

			if node, ok := nodes[nodeID]; ok {
				node.Get("style").Call("setProperty", prop, val)
			}

		case 0x08: // OpPushHistory
			path := string(data[pos:])
			pos = len(data)
			js.Global().Get("history").Call("pushState", nil, "", path)

		default:
			fmt.Printf("cbs: unknown opcode 0x%02x\n", op)
			return
		}
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
