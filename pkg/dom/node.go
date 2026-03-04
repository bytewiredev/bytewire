package dom

import "sync/atomic"

// NodeID uniquely identifies a DOM node within a session.
type NodeID uint32

var nodeCounter atomic.Uint32

func nextNodeID() NodeID {
	return NodeID(nodeCounter.Add(1))
}

// NodeType distinguishes element nodes from text nodes.
type NodeType byte

const (
	ElementNode NodeType = 1
	TextNode    NodeType = 2
)

// Node represents a virtual DOM node managed by the server.
type Node struct {
	ID       NodeID
	Type     NodeType
	Tag      string
	Attrs    map[string]string
	Styles   map[string]string
	Text     string
	Children []*Node
	Parent   *Node

	// Event handlers keyed by event type byte (EventClick, etc.)
	Handlers map[byte]func([]byte)

	// Dirty is set to true by signal observers when the node's text changes.
	Dirty bool
	// SignalBound is true if this node was created via TextF and is bound to a signal.
	SignalBound bool
}

// newElement creates an element node with the given tag.
func newElement(tag string) *Node {
	return &Node{
		ID:       nextNodeID(),
		Type:     ElementNode,
		Tag:      tag,
		Attrs:    make(map[string]string),
		Styles:   make(map[string]string),
		Handlers: make(map[byte]func([]byte)),
	}
}

// newText creates a text node.
func newText(text string) *Node {
	return &Node{
		ID:   nextNodeID(),
		Type: TextNode,
		Text: text,
	}
}

// AppendChild adds a child node.
func (n *Node) AppendChild(child *Node) *Node {
	child.Parent = n
	n.Children = append(n.Children, child)
	return n
}
