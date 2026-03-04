package dom

import "github.com/cbsframework/cbs/pkg/protocol"

// Diff compares an old and new node tree and emits binary opcodes
// representing the minimal set of DOM mutations needed.
func Diff(buf *protocol.Buffer, old, next *Node) {
	if old == nil && next == nil {
		return
	}

	// New node inserted
	if old == nil && next != nil {
		emitInsert(buf, next)
		return
	}

	// Node removed
	if old != nil && next == nil {
		buf.EncodeRemoveNode(uint32(old.ID))
		return
	}

	// Both exist — diff them
	if old.Type == TextNode && next.Type == TextNode {
		if old.Text != next.Text {
			buf.EncodeUpdateText(uint32(old.ID), next.Text)
		}
		return
	}

	// Diff attributes
	for k, v := range next.Attrs {
		if old.Attrs[k] != v {
			buf.EncodeSetAttr(uint32(old.ID), k, v)
		}
	}
	for k := range old.Attrs {
		if _, exists := next.Attrs[k]; !exists {
			buf.EncodeRemoveAttr(uint32(old.ID), k)
		}
	}

	// Diff inline styles
	for k, v := range next.Styles {
		if old.Styles[k] != v {
			buf.EncodeSetStyle(uint32(old.ID), k, v)
		}
	}

	// Diff children
	maxLen := len(old.Children)
	if len(next.Children) > maxLen {
		maxLen = len(next.Children)
	}
	for i := range maxLen {
		var oldChild, nextChild *Node
		if i < len(old.Children) {
			oldChild = old.Children[i]
		}
		if i < len(next.Children) {
			nextChild = next.Children[i]
		}
		Diff(buf, oldChild, nextChild)
	}
}

// emitInsert encodes a full node insertion, recursing into children.
func emitInsert(buf *protocol.Buffer, n *Node) {
	if n.Type == TextNode {
		parentID := uint32(0)
		if n.Parent != nil {
			parentID = uint32(n.Parent.ID)
		}
		buf.EncodeInsertNode(uint32(n.ID), parentID, 0, "#text", nil)
		buf.EncodeUpdateText(uint32(n.ID), n.Text)
		return
	}

	parentID := uint32(0)
	if n.Parent != nil {
		parentID = uint32(n.Parent.ID)
	}
	buf.EncodeInsertNode(uint32(n.ID), parentID, 0, n.Tag, n.Attrs)

	for _, child := range n.Children {
		emitInsert(buf, child)
	}
}
