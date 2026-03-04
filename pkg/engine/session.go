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

	currentPath string
	navHandler  func(path string)
	routeParams map[string]string
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

// HandleIntent processes a client message (intent or navigation) by dispatching
// to the appropriate handler, then flushing any resulting DOM updates.
func (s *Session) HandleIntent(data []byte) error {
	msg, _, err := protocol.DecodeFrame(data)
	if err != nil {
		return err
	}

	switch msg.Op {
	case protocol.OpClientIntent:
		return s.handleClientIntent(msg)
	case protocol.OpClientNav:
		return s.handleClientNav(msg.Text)
	default:
		return nil
	}
}

// handleClientIntent dispatches a user interaction event to the target node's handler.
func (s *Session) handleClientIntent(msg protocol.Message) error {
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

	handler(msg.Payload)

	return s.flushDirtyNodes()
}

// handleClientNav processes a client-side navigation event.
func (s *Session) handleClientNav(path string) error {
	if s.navHandler == nil {
		s.logger.Warn("nav event but no handler registered", "path", path)
		return nil
	}

	s.currentPath = path
	s.navHandler(path)

	return s.flushDirtyNodes()
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

// CurrentPath returns the session's current navigation path.
func (s *Session) CurrentPath() string {
	return s.currentPath
}

// SetCurrentPath sets the session's current navigation path.
func (s *Session) SetCurrentPath(path string) {
	s.currentPath = path
}

// SetNavHandler registers a callback invoked on client navigation events.
func (s *Session) SetNavHandler(fn func(path string)) {
	s.navHandler = fn
}

// SetRouteParams stores the current route's extracted parameters.
func (s *Session) SetRouteParams(params map[string]string) {
	s.routeParams = params
}

// RouteParam returns a named route parameter value.
func (s *Session) RouteParam(key string) string {
	if s.routeParams == nil {
		return ""
	}
	return s.routeParams[key]
}

// emitFullTree serializes the entire DOM tree as InsertNode + UpdateText opcodes.
func emitFullTree(buf *protocol.Buffer, n *dom.Node) {
	if n == nil {
		return
	}
	if n.Type == dom.TextNode {
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
		emitFullTree(buf, child)
	}
}

// flushDirtyNodes walks the tree, drains PendingOps (structural changes),
// emits OpUpdateText for dirty text, and emits OpSetAttr/OpSetStyle for
// dirty attributes and styles.
func (s *Session) flushDirtyNodes() error {
	s.mu.Lock()
	var dirty []*dom.Node
	collectDirty(s.root, &dirty)
	s.mu.Unlock()

	if len(dirty) == 0 {
		return nil
	}

	buf := protocol.AcquireBuffer()
	defer buf.Release()

	for _, n := range dirty {
		// 1. Drain structural PendingOps first (insert/remove)
		for _, op := range n.PendingOps {
			op(buf)
		}
		n.PendingOps = n.PendingOps[:0]

		// 2. Emit text updates
		if n.DirtyText {
			buf.EncodeUpdateText(uint32(n.ID), n.Text)
			n.DirtyText = false
		}

		// 3. Emit attribute updates
		for key, val := range n.DirtyAttrs {
			if val == "" {
				buf.EncodeRemoveAttr(uint32(n.ID), key)
			} else {
				buf.EncodeSetAttr(uint32(n.ID), key, val)
			}
		}
		clear(n.DirtyAttrs)

		// 4. Emit style updates
		for prop, val := range n.DirtyStyles {
			buf.EncodeSetStyle(uint32(n.ID), prop, val)
		}
		clear(n.DirtyStyles)

		n.Dirty = false
	}

	return s.writer.WriteMessage(buf.Bytes())
}

// collectDirty gathers all nodes with Dirty=true.
func collectDirty(n *dom.Node, out *[]*dom.Node) {
	if n == nil {
		return
	}
	if n.Dirty {
		*out = append(*out, n)
	}
	for _, child := range n.Children {
		collectDirty(child, out)
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
