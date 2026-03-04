// Package engine provides the CBS server runtime: WebTransport listener,
// per-user session goroutines, and binary stream multiplexing.
package engine

import (
	"context"
	"log/slog"
	"sync"

	"github.com/cbsframework/cbs/pkg/dom"
	"github.com/cbsframework/cbs/pkg/protocol"
)

// SessionID uniquely identifies a connected user session.
type SessionID uint64

// Component is a function that returns the root DOM node for a session.
// It receives the session so it can register signals and event handlers.
type Component func(s *Session) *dom.Node

// Session manages the state for a single connected user.
// Each session runs in its own goroutine with its own virtual DOM tree.
type Session struct {
	ID     SessionID
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	root   *dom.Node
	writer Writer
	logger *slog.Logger
}

// Writer abstracts the binary transport. The engine writes opcode frames here.
type Writer interface {
	WriteMessage(data []byte) error
	Close() error
}

var sessionCounter uint64
var sessionMu sync.Mutex

func nextSessionID() SessionID {
	sessionMu.Lock()
	sessionCounter++
	id := sessionCounter
	sessionMu.Unlock()
	return SessionID(id)
}

// NewSession creates a session bound to a transport writer.
func NewSession(ctx context.Context, w Writer, logger *slog.Logger) *Session {
	ctx, cancel := context.WithCancel(ctx)
	return &Session{
		ID:     nextSessionID(),
		ctx:    ctx,
		cancel: cancel,
		writer: w,
		logger: logger,
	}
}

// Mount renders the component and sends the initial DOM tree as binary opcodes.
func (s *Session) Mount(comp Component) error {
	s.mu.Lock()
	s.root = comp(s)
	s.mu.Unlock()

	buf := protocol.AcquireBuffer()
	defer buf.Release()

	emitFullTree(buf, s.root)
	return s.writer.WriteMessage(buf.Bytes())
}

// HandleIntent processes a client event (click, input, etc.) by dispatching
// to the appropriate node handler, then diffing and sending any resulting updates.
func (s *Session) HandleIntent(data []byte) error {
	msg, _, err := protocol.Decode(data)
	if err != nil {
		return err
	}

	if msg.Op != protocol.OpClientIntent {
		return nil
	}

	s.mu.Lock()
	node := findNode(s.root, dom.NodeID(msg.NodeID))
	s.mu.Unlock()

	if node == nil {
		s.logger.Warn("intent for unknown node", "nodeID", msg.NodeID)
		return nil
	}

	handler, ok := node.Handlers[msg.EventType]
	if !ok {
		return nil
	}

	// Snapshot old tree for diffing
	// For now, we rely on signal observers to mutate the tree in-place
	// and then re-emit changed nodes.
	handler(msg.Payload)

	return nil
}

// Close terminates the session.
func (s *Session) Close() {
	s.cancel()
	s.writer.Close()
}

// Context returns the session's context.
func (s *Session) Context() context.Context {
	return s.ctx
}

// emitFullTree serializes the entire DOM tree as InsertNode + UpdateText opcodes.
func emitFullTree(buf *protocol.Buffer, n *dom.Node) {
	if n == nil {
		return
	}
	if n.Type == dom.TextNode {
		buf.EncodeUpdateText(uint32(n.ID), n.Text)
		return
	}

	parentID := uint32(0)
	if n.Parent != nil {
		parentID = uint32(n.Parent.ID)
	}
	buf.EncodeInsertNode(parentID, 0, n.Tag, n.Attrs)

	for _, child := range n.Children {
		emitFullTree(buf, child)
	}
}

// findNode does a DFS search for a node by ID.
func findNode(root *dom.Node, id dom.NodeID) *dom.Node {
	if root == nil {
		return nil
	}
	if root.ID == id {
		return root
	}
	for _, child := range root.Children {
		if found := findNode(child, id); found != nil {
			return found
		}
	}
	return nil
}
