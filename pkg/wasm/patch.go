//go:build js && wasm

package wasm

import (
	"encoding/binary"
	"fmt"
	"syscall/js"
)

// Batch state for deferred DOM attachment. When processing a batch of opcodes,
// new nodes whose parent is already in the live DOM are collected and attached
// via DocumentFragment at the end, reducing live DOM mutations from O(n) to O(1)
// per parent container.
var (
	batchDepth    int
	batchExisting map[uint32]bool  // node IDs that existed before this batch
	batchDeferred []deferredInsert // insertions to defer
)

type deferredInsert struct {
	nodeID, parentID uint32
}

// applyOpcodes processes a binary opcode stream and applies DOM mutations.
// Each opcode is wrapped in a 4-byte length-prefixed frame.
func applyOpcodes(data []byte) {
	batchDepth++
	if batchDepth == 1 {
		// Snapshot live nodes before this batch
		batchExisting = make(map[uint32]bool, len(nodes))
		for id := range nodes {
			batchExisting[id] = true
		}
		batchDeferred = batchDeferred[:0]
	}

	pos := 0
	for pos < len(data) {
		// Read frame length prefix
		if pos+4 > len(data) {
			break
		}
		frameLen := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if pos+frameLen > len(data) {
			break
		}
		frame := data[pos : pos+frameLen]
		pos += frameLen

		applyFrame(frame)
	}

	batchDepth--
	if batchDepth == 0 {
		flushDeferredInserts()
		updateDevToolsNodeCount()
	}
}

// flushDeferredInserts attaches deferred nodes to their live parents using
// DocumentFragment for batching, resulting in a single DOM mutation per parent.
func flushDeferredInserts() {
	if len(batchDeferred) == 0 {
		return
	}

	type fragEntry struct {
		parentID uint32
		frag     js.Value
	}
	fragMap := make(map[uint32]int) // parentID -> index in frags
	var frags []fragEntry

	for _, d := range batchDeferred {
		idx, ok := fragMap[d.parentID]
		if !ok {
			frag := document.Call("createDocumentFragment")
			idx = len(frags)
			frags = append(frags, fragEntry{d.parentID, frag})
			fragMap[d.parentID] = idx
		}
		if node, ok := nodes[d.nodeID]; ok {
			frags[idx].frag.Call("appendChild", node)
		}
	}

	for _, fe := range frags {
		if parent, ok := nodes[fe.parentID]; ok {
			parent.Call("appendChild", fe.frag)
		} else if fe.parentID == 0 {
			root.Call("appendChild", fe.frag)
		}
	}

	batchDeferred = batchDeferred[:0]
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
			setAttrFast(node, key, value)
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
		siblingID := binary.BigEndian.Uint32(data[p : p+4])
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

		// Move semantics: if nodeID already exists, detach and reattach.
		if existing, ok := nodes[nodeID]; ok {
			oldParent := existing.Get("parentNode")
			if !oldParent.IsNull() && !oldParent.IsUndefined() {
				oldParent.Call("removeChild", existing)
			}

			// Skip attribute parsing for moves
			for range attrCount {
				if p+2 > len(data) {
					return
				}
				kLen := int(binary.BigEndian.Uint16(data[p : p+2]))
				p += 2
				p += kLen
				if p+2 > len(data) {
					return
				}
				vLen := int(binary.BigEndian.Uint16(data[p : p+2]))
				p += 2
				p += vLen
			}

			// Reattach at new position
			parent := root
			if pp, ok := nodes[parentID]; ok {
				parent = pp
			}
			if siblingID != 0 {
				if sib, ok := nodes[siblingID]; ok {
					parent.Call("insertBefore", existing, sib)
					return
				}
			}
			parent.Call("appendChild", existing)
			return
		}

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

			setAttrFast(el, k, v)
		}

		// Set __bwId as a JS property (faster than setAttribute + avoids string formatting).
		if tag != "#text" {
			el.Set("__bwId", nodeID)
		}

		nodes[nodeID] = el

		// Defer attachment if parent is in the live DOM and we're appending
		// (no sibling). This lets us batch all children into a DocumentFragment
		// and do a single DOM mutation per parent at flush time.
		if siblingID == 0 && batchDepth > 0 && batchExisting[parentID] {
			batchDeferred = append(batchDeferred, deferredInsert{nodeID, parentID})
		} else if parent, ok := nodes[parentID]; ok {
			if siblingID != 0 {
				if sib, ok := nodes[siblingID]; ok {
					parent.Call("insertBefore", el, sib)
					return
				}
			}
			parent.Call("appendChild", el)
		} else if parentID == 0 {
			root.Call("appendChild", el)
		}

	case 0x0E: // OpInsertText — combined create text node + set content
		if p+8 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		parentID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		text := string(data[p:])

		el := document.Call("createTextNode", text)
		nodes[nodeID] = el

		if batchDepth > 0 && batchExisting[parentID] {
			batchDeferred = append(batchDeferred, deferredInsert{nodeID, parentID})
		} else if parent, ok := nodes[parentID]; ok {
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
			// Recursively clean up descendant entries from the nodes map.
			cleanupDescendants(node)
			parent := node.Get("parentNode")
			if !parent.IsNull() && !parent.IsUndefined() {
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

	case 0x06: // OpReplaceText
		if p+12 > len(data) {
			return
		}
		nodeID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		offset := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		length := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		replacement := string(data[p:])

		if node, ok := nodes[nodeID]; ok {
			// Splice text at byte offsets. Assumes UTF-8 aligned offsets from server.
			current := node.Get("textContent").String()
			end := min(int(offset+length), len(current))
			if int(offset) > len(current) {
				return
			}
			newText := current[:offset] + replacement + current[end:]
			node.Set("textContent", newText)
		}

	case 0x08: // OpPushHistory
		path := string(data[p:])
		js.Global().Get("history").Call("pushState", nil, "", path)

	case 0x0A: // OpError
		if p+2 > len(data) {
			return
		}
		msgLen := int(binary.BigEndian.Uint16(data[p : p+2]))
		p += 2
		if p+msgLen > len(data) {
			return
		}
		errMsg := string(data[p : p+msgLen])
		showErrorOverlay(errMsg)

	case 0x0B: // OpDevToolsState
		handleDevToolsState(data)

	case 0x09: // OpBatch
		if p+4 > len(data) {
			return
		}
		// Skip 4-byte count — applyOpcodes iterates by length prefix naturally.
		// The outer frame already bounds data, so data[p+4:] contains exactly
		// the nested length-prefixed frames.
		applyOpcodes(data[p+4:])

	case 0x00: // OpHello
		handleHello(data)

	case 0x0C: // OpAuthChallenge
		CompletePasskey(data)

	case 0x0D: // OpAuthResult
		handleAuthResult(data)

	case 0x0F: // OpInsertHTML
		if p+4 > len(data) {
			return
		}
		parentID := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		htmlStr := string(data[p:])

		// Resolve parent element
		parent := root
		if pp, ok := nodes[parentID]; ok {
			parent = pp
		}

		// Call the pure-JS helper — does template creation, innerHTML parsing,
		// querySelectorAll walk, __bwId setting, and appendChild entirely in JS.
		// Returns a Uint8Array of packed [uint32 id, int32 tid] pairs.
		idBuf := js.Global().Call("__bwProcessHTML", parent, htmlStr)

		// Bulk-copy the ID buffer to Go (1 interop call instead of ~36000).
		bufLen := idBuf.Get("length").Int()
		if bufLen > 0 {
			idBytes := make([]byte, bufLen)
			js.CopyBytesToGo(idBytes, idBuf)

			els := js.Global().Get("__bwHTMLEls")
			count := bufLen / 8
			for i := 0; i < count; i++ {
				id := binary.LittleEndian.Uint32(idBytes[i*8:])
				tidRaw := int32(binary.LittleEndian.Uint32(idBytes[i*8+4:]))
				el := els.Call("item", i)
				nodes[id] = el
				if tidRaw >= 0 {
					nodes[uint32(tidRaw)] = el
				}
				// Trigger initial chart render for canvas elements.
				chartAttr := el.Call("getAttribute", "data-bw-chart")
				if !chartAttr.IsNull() && !chartAttr.IsUndefined() {
					renderChart(el)
				}
			}
		}

	case 0x15: // OpSwapNodes
		if p+8 > len(data) {
			return
		}
		nodeA := binary.BigEndian.Uint32(data[p : p+4])
		p += 4
		nodeB := binary.BigEndian.Uint32(data[p : p+4])

		elA, okA := nodes[nodeA]
		elB, okB := nodes[nodeB]
		if !okA || !okB {
			return
		}

		// DOM swap: save A's next sibling, move A before B, move B to A's old position.
		parentA := elA.Get("parentNode")
		nextA := elA.Get("nextSibling")

		// If B is directly after A, insertBefore(B, A) is sufficient.
		if nextA.Equal(elB) {
			parentA.Call("insertBefore", elB, elA)
		} else {
			// General case: move A before B, then move B to A's old spot.
			parentB := elB.Get("parentNode")
			parentB.Call("insertBefore", elA, elB)
			if nextA.IsNull() || nextA.IsUndefined() {
				parentA.Call("appendChild", elB)
			} else {
				parentA.Call("insertBefore", elB, nextA)
			}
		}

	case 0x16: // OpBatchText
		if p+2 > len(data) {
			return
		}
		count := int(binary.BigEndian.Uint16(data[p : p+2]))
		p += 2
		for range count {
			if p+6 > len(data) {
				return
			}
			nodeID := binary.BigEndian.Uint32(data[p : p+4])
			p += 4
			textLen := int(binary.BigEndian.Uint16(data[p : p+2]))
			p += 2
			if p+textLen > len(data) {
				return
			}
			text := string(data[p : p+textLen])
			p += textLen
			if node, ok := nodes[nodeID]; ok {
				node.Set("textContent", text)
			}
		}

	case 0x14: // OpClearChildren
		if p+4 > len(data) {
			return
		}
		parentID := binary.BigEndian.Uint32(data[p : p+4])

		var parent js.Value
		if pp, ok := nodes[parentID]; ok {
			parent = pp
		} else {
			parent = root
		}

		// Collect all descendant __bwId values via JS helper, then
		// bulk-transfer as a Uint8Array (one CopyBytesToGo call).
		idBuf := js.Global().Call("__bwCollectIds", parent)
		bufLen := idBuf.Get("length").Int()
		if bufLen > 0 {
			idBytes := make([]byte, bufLen)
			js.CopyBytesToGo(idBytes, idBuf)
			for i := 0; i < bufLen; i += 4 {
				id := binary.LittleEndian.Uint32(idBytes[i:])
				delete(nodes, id)
			}
		}

		// Remove all children in one DOM operation.
		parent.Set("textContent", "")

	default:
		fmt.Printf("bytewire: unknown opcode 0x%02x\n", op)
	}
}

func handleAuthResult(data []byte) {
	if len(data) < 2 {
		return
	}
	success := data[1] == 1
	token := ""
	if len(data) > 2 {
		token = string(data[2:])
	}
	if success {
		fmt.Printf("bytewire: authenticated (token=%s)\n", token)
	} else {
		fmt.Println("bytewire: authentication failed")
	}
}

// parseDataBwId parses a numeric string to uint32 without allocations.
func parseDataBwId(s string) uint32 {
	var id uint32
	for _, c := range s {
		if c >= '0' && c <= '9' {
			id = id*10 + uint32(c-'0')
		}
	}
	return id
}

// setAttrFast uses direct property assignment for common attributes (id, className)
// which is faster than setAttribute because it skips DOM attribute mutation overhead.
func setAttrFast(el js.Value, k, v string) {
	switch k {
	case "id":
		el.Set("id", v)
	case "class":
		el.Set("className", v)
	case "data-bw-chart-data":
		el.Call("setAttribute", k, v)
		renderChart(el)
	default:
		el.Call("setAttribute", k, v)
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

// showErrorOverlay renders a fixed-position red error banner at the top of the page.
// It auto-dismisses after 10 seconds and includes a dismiss button.
func showErrorOverlay(msg string) {
	overlay := document.Call("createElement", "div")
	overlay.Call("setAttribute", "style",
		"position:fixed;top:0;left:0;right:0;z-index:99999;"+
			"background:#dc2626;color:#fff;padding:12px 16px;"+
			"font-family:monospace;font-size:14px;line-height:1.4;"+
			"display:flex;align-items:flex-start;gap:12px;",
	)

	msgSpan := document.Call("createElement", "span")
	msgSpan.Call("setAttribute", "style", "flex:1;white-space:pre-wrap;word-break:break-word;")
	msgSpan.Set("textContent", msg)
	overlay.Call("appendChild", msgSpan)

	btn := document.Call("createElement", "button")
	btn.Call("setAttribute", "style",
		"background:none;border:none;color:#fff;font-size:20px;"+
			"cursor:pointer;padding:0;line-height:1;flex-shrink:0;",
	)
	btn.Set("textContent", "\u00d7")

	dismiss := js.FuncOf(func(this js.Value, args []js.Value) any {
		parent := overlay.Get("parentNode")
		if !parent.IsNull() && !parent.IsUndefined() {
			parent.Call("removeChild", overlay)
		}
		return nil
	})
	btn.Call("addEventListener", "click", dismiss)
	overlay.Call("appendChild", btn)

	document.Get("body").Call("appendChild", overlay)

	// Auto-dismiss after 10 seconds.
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		parent := overlay.Get("parentNode")
		if !parent.IsNull() && !parent.IsUndefined() {
			parent.Call("removeChild", overlay)
		}
		return nil
	}), 10000)
}

// hydrateExistingDOM scans the DOM for elements with data-bw-id attributes
// (from SSR) and pre-populates the nodes map. It also sets the __bwId JS
// property so event delegation uses the fast property path.
func hydrateExistingDOM() {
	nodeList := document.Call("querySelectorAll", "[data-bw-id]")
	length := nodeList.Get("length").Int()
	if length == 0 {
		return
	}

	for i := 0; i < length; i++ {
		el := nodeList.Call("item", i)
		attr := el.Call("getAttribute", "data-bw-id")
		if attr.IsNull() || attr.IsUndefined() {
			continue
		}
		idStr := attr.String()
		var id uint32
		for _, c := range idStr {
			if c >= '0' && c <= '9' {
				id = id*10 + uint32(c-'0')
			}
		}
		if id > 0 {
			nodes[id] = el
			el.Set("__bwId", id) // fast property for event delegation
		}
	}

	fmt.Printf("bytewire: hydrated %d SSR nodes\n", length)
}

// cleanupDescendants removes all descendant nodes from the nodes map.
func cleanupDescendants(el js.Value) {
	children := el.Get("children")
	if children.IsUndefined() || children.IsNull() {
		return
	}
	length := children.Get("length").Int()
	for i := range length {
		child := children.Index(i)
		cleanupDescendants(child)
		bwID := child.Get("__bwId")
		if !bwID.IsUndefined() && !bwID.IsNull() {
			delete(nodes, uint32(bwID.Int()))
		}
	}
}
