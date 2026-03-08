package dom

import (
	"html"
	"strconv"
	"strings"

	"github.com/bytewiredev/bytewire/pkg/protocol"
)

// If conditionally renders one of two subtrees based on a signal's value.
// The returned container node is a stable anchor in the DOM.
// When the condition flips, the old subtree is removed and the new one inserted.
// The els parameter is optional — pass nil to render nothing when the condition is false.
func If[T any](s Observable[T], cond func(T) bool, then func() *Node, els func() *Node) *Node {
	container := newElement("div")
	active := cond(s.Get())
	var current *Node

	if active {
		current = then()
		container.AppendChild(current)
	} else if els != nil {
		current = els()
		container.AppendChild(current)
	}

	s.Observe(func(v T) {
		newActive := cond(v)
		if newActive == active {
			return
		}
		active = newActive

		oldChild := current
		current = nil

		container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
			// Remove old subtree
			if oldChild != nil {
				buf.EncodeRemoveNode(uint32(oldChild.ID))
			}
		})

		if oldChild != nil {
			container.RemoveChild(oldChild)
		}

		var newChild *Node
		if newActive {
			newChild = then()
		} else if els != nil {
			newChild = els()
		}

		if newChild != nil {
			container.AppendChild(newChild)
			current = newChild
			emitInsertOps(container, newChild)
		}

		container.Dirty = true
	})

	return container
}

// For renders a list of items from a ListSignal using keyed reconciliation.
// keyFn extracts a unique key from each item for efficient diffing.
// render creates a DOM node for each item.
func For[T any](s *ListSignal[T], keyFn func(T) string, render func(T) *Node) *Node {
	container := newElement("div")
	nodeMap := make(map[string]*Node) // key -> node

	// Initial render
	items := s.Get()
	for _, item := range items {
		key := keyFn(item)
		child := render(item)
		nodeMap[key] = child
		container.AppendChild(child)
	}

	s.Observe(func(newItems []T) {
		// Build new key set
		newKeys := make(map[string]bool, len(newItems))
		for _, item := range newItems {
			newKeys[keyFn(item)] = true
		}

		// Remove disappeared keys
		if len(newKeys) == 0 && len(nodeMap) > 0 {
			// All items removed — emit single OpClearChildren instead of N removes.
			parentID := uint32(container.ID)
			container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
				buf.EncodeClearChildren(parentID)
			})
			container.Children = container.Children[:0]
			for key := range nodeMap {
				delete(nodeMap, key)
			}
		} else {
			for key, node := range nodeMap {
				if !newKeys[key] {
					removedNode := node
					container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
						buf.EncodeRemoveNode(uint32(removedNode.ID))
					})
					container.RemoveChild(node)
					delete(nodeMap, key)
				}
			}
		}

		// Build old key→position index for existing (surviving) items.
		nodeToKey := make(map[*Node]string, len(nodeMap))
		for key, node := range nodeMap {
			nodeToKey[node] = key
		}
		oldKeyPos := make(map[string]int, len(container.Children))
		for i, child := range container.Children {
			if key, ok := nodeToKey[child]; ok {
				oldKeyPos[key] = i
			}
		}

		// Build the sequence of old positions in new order (existing items only).
		oldPositions := make([]int, 0, len(newItems))
		for _, item := range newItems {
			key := keyFn(item)
			if pos, ok := oldKeyPos[key]; ok {
				oldPositions = append(oldPositions, pos)
			}
		}

		// Compute LIS to find the largest set of items already in order.
		// Items NOT in the LIS are the minimum set that need move opcodes.
		lisSet := lisIndices(oldPositions)

		// Fast path: detect exact 2-element swap (no adds/removes, exactly 2 out of order).
		if len(oldPositions) == len(newItems) && len(oldPositions) >= 2 {
			outOfOrder := 0
			var swapA, swapB int
			for i, inLIS := range lisSet {
				if !inLIS {
					if outOfOrder == 0 {
						swapA = i
					} else if outOfOrder == 1 {
						swapB = i
					}
					outOfOrder++
					if outOfOrder > 2 {
						break
					}
				}
			}
			if outOfOrder == 2 {
				// Emit a single OpSwapNodes — much cheaper than 2 OpInsertNode moves.
				keyA := keyFn(newItems[swapA])
				keyB := keyFn(newItems[swapB])
				nodeA := nodeMap[keyA]
				nodeB := nodeMap[keyB]
				idA := uint32(nodeA.ID)
				idB := uint32(nodeB.ID)
				container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
					buf.EncodeSwapNodes(idA, idB)
				})
				// Rebuild children slice in new order
				container.Children = container.Children[:0]
				for _, item := range newItems {
					key := keyFn(item)
					node := nodeMap[key]
					node.Parent = container
					container.Children = append(container.Children, node)
				}
				container.Dirty = true
				return
			}
		}

		// Rebuild children in new order
		container.Children = container.Children[:0]
		var bulkNew []*Node
		existIdx := 0
		for _, item := range newItems {
			key := keyFn(item)
			if existing, ok := nodeMap[key]; ok {
				if !lisSet[existIdx] {
					// This item needs a move — flush pending bulk new first
					if len(bulkNew) > 0 {
						flushBulkNew(container, bulkNew)
						bulkNew = bulkNew[:0]
					}
					movedNode := existing
					parentID := uint32(container.ID)
					container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
						buf.EncodeInsertNode(uint32(movedNode.ID), parentID, 0, movedNode.Tag, movedNode.Attrs)
					})
				}
				existIdx++
				existing.Parent = container
				container.Children = append(container.Children, existing)
			} else {
				// New item — collect for potential bulk insert
				child := render(item)
				nodeMap[key] = child
				child.Parent = container
				container.Children = append(container.Children, child)
				bulkNew = append(bulkNew, child)
			}
		}
		// Flush remaining new items
		if len(bulkNew) > 0 {
			flushBulkNew(container, bulkNew)
		}

		container.Dirty = true
	})

	return container
}

// flushBulkNew emits insert operations for a batch of new children.
// Uses HTML bulk insert when above threshold, individual ops otherwise.
func flushBulkNew(container *Node, children []*Node) {
	if len(children) >= bulkInsertThreshold {
		emitBulkInsertHTML(container, children)
	} else {
		for _, child := range children {
			emitInsertOps(container, child)
		}
	}
}

// emitInsertOps queues PendingOps to emit the full insert subtree for a newly added child.
func emitInsertOps(parent *Node, child *Node) {
	parent.PendingOps = append(parent.PendingOps, func(buf *protocol.Buffer) {
		emitInsertTree(buf, child)
	})
}

// QueueInsert queues PendingOps on the parent to emit the full insert subtree
// for a child node. This is the public API used by the router and other packages
// that need to dynamically insert subtrees.
func QueueInsert(parent *Node, child *Node) {
	emitInsertOps(parent, child)
}

// emitInsertTree recursively emits OpInsertNode/OpInsertText for a subtree.
func emitInsertTree(buf *protocol.Buffer, n *Node) {
	if n.Type == TextNode {
		parentID := uint32(0)
		if n.Parent != nil {
			parentID = uint32(n.Parent.ID)
		}
		buf.EncodeInsertText(uint32(n.ID), parentID, n.Text)
		return
	}

	parentID := uint32(0)
	if n.Parent != nil {
		parentID = uint32(n.Parent.ID)
	}
	buf.EncodeInsertNode(uint32(n.ID), parentID, 0, n.Tag, n.Attrs)

	for _, child := range n.Children {
		emitInsertTree(buf, child)
	}
}

// bulkInsertThreshold is the minimum number of new children to trigger
// HTML bulk insertion instead of individual opcodes.
const bulkInsertThreshold = 20

// emitBulkInsertHTML renders multiple children as a single HTML string
// and emits an OpInsertHTML opcode. This uses the browser's native HTML
// parser which is significantly faster than individual createElement calls.
func emitBulkInsertHTML(parent *Node, children []*Node) {
	var b strings.Builder
	for _, child := range children {
		renderNodeForInsert(&b, child)
	}
	htmlStr := b.String()
	parent.PendingOps = append(parent.PendingOps, func(buf *protocol.Buffer) {
		buf.EncodeInsertHTML(uint32(parent.ID), htmlStr)
	})
}

// lisIndices computes the Longest Increasing Subsequence of the given
// positions and returns a boolean slice where true means the item at that
// index is part of the LIS (i.e., does NOT need a move operation).
func lisIndices(positions []int) []bool {
	n := len(positions)
	if n == 0 {
		return nil
	}

	// tails[i] = smallest tail value of an increasing subsequence of length i+1
	tails := make([]int, 0, n)
	// tailIdx[i] = index in positions[] of the value stored in tails[i]
	tailIdx := make([]int, 0, n)
	// prev[i] = index of previous element in the LIS ending at positions[i]
	prev := make([]int, n)
	for i := range prev {
		prev[i] = -1
	}

	for i, val := range positions {
		// Binary search for the leftmost tail >= val
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := (lo + hi) / 2
			if tails[mid] < val {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo == len(tails) {
			tails = append(tails, val)
			tailIdx = append(tailIdx, i)
		} else {
			tails[lo] = val
			tailIdx[lo] = i
		}
		if lo > 0 {
			prev[i] = tailIdx[lo-1]
		}
	}

	// Reconstruct which indices are in the LIS
	inLIS := make([]bool, n)
	k := tailIdx[len(tailIdx)-1]
	for k >= 0 {
		inLIS[k] = true
		k = prev[k]
	}
	return inLIS
}

// renderNodeForInsert renders a node subtree as HTML for OpInsertHTML.
// Signal-bound text nodes are tracked via a data-bw-tid attribute on their
// parent element, avoiding extra wrapper elements in the DOM.
func renderNodeForInsert(b *strings.Builder, n *Node) {
	if n.Type == TextNode {
		b.WriteString(html.EscapeString(n.Text))
		return
	}

	b.WriteString("<")
	b.WriteString(n.Tag)
	b.WriteString(` data-bw-id="`)
	b.WriteString(strconv.FormatUint(uint64(n.ID), 10))
	b.WriteByte('"')

	// If this element has a signal-bound text child, add data-bw-tid so the
	// client registers the element as a proxy for OpUpdateText targeting that text node.
	for _, child := range n.Children {
		if child.Type == TextNode && child.SignalBound {
			b.WriteString(` data-bw-tid="`)
			b.WriteString(strconv.FormatUint(uint64(child.ID), 10))
			b.WriteByte('"')
			break
		}
	}

	for k, v := range n.Attrs {
		b.WriteByte(' ')
		b.WriteString(html.EscapeString(k))
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(v))
		b.WriteByte('"')
	}
	if len(n.Styles) > 0 {
		b.WriteString(` style="`)
		first := true
		for k, v := range n.Styles {
			if !first {
				b.WriteByte(';')
			}
			b.WriteString(k)
			b.WriteByte(':')
			b.WriteString(v)
			first = false
		}
		b.WriteByte('"')
	}
	b.WriteString(">")

	for _, child := range n.Children {
		renderNodeForInsert(b, child)
	}

	b.WriteString("</")
	b.WriteString(n.Tag)
	b.WriteString(">")
}
