package dom

import "github.com/cbsframework/cbs/pkg/protocol"

// If conditionally renders one of two subtrees based on a signal's value.
// The returned container node is a stable anchor in the DOM.
// When the condition flips, the old subtree is removed and the new one inserted.
// The els parameter is optional — pass nil to render nothing when the condition is false.
func If[T comparable](s *Signal[T], cond func(T) bool, then func() *Node, els func() *Node) *Node {
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

		// Rebuild children in new order, creating new nodes as needed
		container.Children = container.Children[:0]
		for _, item := range newItems {
			key := keyFn(item)
			if existing, ok := nodeMap[key]; ok {
				// Reuse existing node — emit move (insert existing ID = move)
				existing.Parent = container
				container.Children = append(container.Children, existing)
				movedNode := existing
				parentID := uint32(container.ID)
				container.PendingOps = append(container.PendingOps, func(buf *protocol.Buffer) {
					buf.EncodeInsertNode(uint32(movedNode.ID), parentID, 0, movedNode.Tag, movedNode.Attrs)
				})
			} else {
				// New item — create and emit full subtree
				child := render(item)
				nodeMap[key] = child
				child.Parent = container
				container.Children = append(container.Children, child)
				emitInsertOps(container, child)
			}
		}

		container.Dirty = true
	})

	return container
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

// emitInsertTree recursively emits OpInsertNode + OpUpdateText for a subtree.
func emitInsertTree(buf *protocol.Buffer, n *Node) {
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
		emitInsertTree(buf, child)
	}
}
