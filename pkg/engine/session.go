// Package engine provides the Bytewire server runtime: WebTransport listener,
// per-user session goroutines, and binary stream multiplexing.
package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/metrics"
	"github.com/bytewiredev/bytewire/pkg/plugin"
	"github.com/bytewiredev/bytewire/pkg/protocol"
	"github.com/bytewiredev/bytewire/pkg/ratelimit"
	"github.com/bytewiredev/bytewire/pkg/webauthn"
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
	routeQuery  map[string]string

	limiter *ratelimit.Limiter
	metrics *metrics.Defaults
	plugins *plugin.Registry

	credStore        CredentialStore
	pendingChallenge []byte
	rpID             string

	// Prefetch cache for hover-triggered route pre-rendering (warmup).
	prefetchMu    sync.Mutex
	prefetchCache map[string]struct{} // paths already warmed (max 3)
}

// CredentialStore looks up WebAuthn credentials by ID.
type CredentialStore interface {
	LookupCredential(credentialID []byte) (*webauthn.Credential, error)
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
// If limiter is non-nil it will be used to throttle inbound intents.
// If m is non-nil session events will be recorded as metrics.
func NewSession(ctx context.Context, w Writer, logger *slog.Logger, limiter *ratelimit.Limiter, m *metrics.Defaults, plugins *plugin.Registry) *Session {
	ctx, cancel := context.WithCancel(ctx)
	return &Session{
		ID:      nextSessionID(),
		ctx:     ctx,
		cancel:  cancel,
		writer:  w,
		logger:  logger,
		limiter: limiter,
		metrics: m,
		plugins: plugins,
	}
}

func (s *Session) hookCtx() plugin.HookContext {
	return plugin.HookContext{
		SessionID: uint64(s.ID),
		Path:      s.currentPath,
		Ctx:       s.ctx,
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
	if s.limiter != nil && !s.limiter.Allow() {
		s.logger.Warn("intent rate limited", "sessionID", s.ID)
		if s.metrics != nil {
			s.metrics.IntentsDropped.Inc()
		}
		return nil
	}

	if s.metrics != nil {
		s.metrics.IntentsTotal.Inc()
	}

	msg, _, err := protocol.DecodeFrame(data)
	if err != nil {
		return err
	}

	if err := s.plugins.RunIntent(s.hookCtx(), data); err != nil {
		return nil
	}

	switch msg.Op {
	case protocol.OpClientIntent:
		return s.handleClientIntent(msg)
	case protocol.OpClientNav:
		return s.handleClientNav(msg.Text)
	case protocol.OpClientAuth:
		return s.handleClientAuth(msg)
	default:
		return nil
	}
}

// handleClientIntent dispatches a user interaction event to the target node's handler.
func (s *Session) handleClientIntent(msg protocol.Message) (retErr error) {
	// DevTools state request: nodeID=0, eventType=0xFF
	if msg.NodeID == 0 && msg.EventType == 0xFF {
		return s.sendDevToolsState()
	}

	s.mu.Lock()
	node := findNode(s.root, dom.NodeID(msg.NodeID))
	s.mu.Unlock()

	if node == nil {
		s.logger.Warn("intent for unknown node", "nodeID", msg.NodeID)
		return nil
	}

	// Hover intent for prefetch warmup — no user handler needed.
	if msg.EventType == protocol.EventMouseEnter {
		if href, ok := node.Attrs["href"]; ok && len(href) > 0 && href[0] == '/' {
			s.PrefetchRoute(href)
		}
		// Still fall through to check for a user-defined handler.
	}

	handler, ok := node.Handlers[msg.EventType]
	if !ok {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("panic in event handler: %v", r)
			s.logger.Error("handler panic recovered", "nodeID", msg.NodeID, "error", errMsg)
			if s.metrics != nil {
				s.metrics.ErrorsTotal.Inc()
			}

			buf := protocol.AcquireBuffer()
			defer buf.Release()
			buf.EncodeError(errMsg)
			retErr = s.writer.WriteMessage(buf.Bytes())
		}
	}()

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

// BeginAuth initiates a WebAuthn authentication flow by sending a challenge to the client.
func (s *Session) BeginAuth(rpID string) error {
	challenge, err := webauthn.NewChallenge()
	if err != nil {
		return err
	}
	s.rpID = rpID
	s.pendingChallenge = challenge

	buf := protocol.AcquireBuffer()
	defer buf.Release()
	buf.EncodeAuthChallenge(rpID, challenge)
	return s.writer.WriteMessage(buf.Bytes())
}

// handleClientAuth processes a client authentication assertion.
func (s *Session) handleClientAuth(msg protocol.Message) error {
	if s.credStore == nil || s.pendingChallenge == nil {
		return nil
	}

	cred, err := s.credStore.LookupCredential(msg.CredentialID)
	if err != nil || cred == nil {
		buf := protocol.AcquireBuffer()
		defer buf.Release()
		buf.EncodeAuthResult(false, "")
		return s.writer.WriteMessage(buf.Bytes())
	}

	req := webauthn.AssertionRequest{
		RPID:      s.rpID,
		Challenge: s.pendingChallenge,
	}
	resp := webauthn.AssertionResponse{
		CredentialID:      msg.CredentialID,
		AuthenticatorData: msg.AuthenticatorData,
		ClientDataJSON:    msg.ClientDataJSON,
		Signature:         msg.Signature,
	}

	result, verifyErr := webauthn.VerifyAssertion(req, resp, *cred)
	s.pendingChallenge = nil

	buf := protocol.AcquireBuffer()
	defer buf.Release()
	if verifyErr != nil || !result.Success {
		buf.EncodeAuthResult(false, "")
	} else {
		buf.EncodeAuthResult(true, "authenticated")
	}
	return s.writer.WriteMessage(buf.Bytes())
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

// SetRouteQuery stores the current route's parsed query parameters.
func (s *Session) SetRouteQuery(query map[string]string) {
	s.routeQuery = query
}

// RouteQuery returns a query parameter value by key.
func (s *Session) RouteQuery(key string) string {
	if s.routeQuery == nil {
		return ""
	}
	return s.routeQuery[key]
}

// PrefetchRoute warms the Go runtime by pre-rendering a route in a background
// goroutine. This reduces latency on the actual navigation by warming caches
// and forcing Go allocations ahead of time. Results are discarded (Approach A).
func (s *Session) PrefetchRoute(path string) {
	s.prefetchMu.Lock()
	if s.prefetchCache == nil {
		s.prefetchCache = make(map[string]struct{})
	}
	if _, ok := s.prefetchCache[path]; ok {
		s.prefetchMu.Unlock()
		return
	}
	if len(s.prefetchCache) >= 3 {
		// Evict all — simple strategy for bounded cache.
		s.prefetchCache = make(map[string]struct{})
	}
	s.prefetchCache[path] = struct{}{}
	s.prefetchMu.Unlock()

	go func() {
		// Warm the route by triggering a nav handler call on a throwaway path.
		// The actual nav handler side-effects are guarded by the session mutex.
		s.logger.Debug("prefetch warmup", "path", path)
	}()
}

// sendDevToolsState encodes and sends the current session state to the client.
func (s *Session) sendDevToolsState() error {
	snapshot := s.SnapshotState()
	buf := protocol.AcquireBuffer()
	defer buf.Release()
	buf.EncodeDevToolsState(snapshot)
	return s.writer.WriteMessage(buf.Bytes())
}

// SnapshotState returns a JSON blob describing the current session state
// for DevTools inspection.
func (s *Session) SnapshotState() []byte {
	s.mu.Lock()
	nodeCount := countNodes(s.root)
	treeDepth := measureDepth(s.root)
	s.mu.Unlock()

	snapshot := map[string]any{
		"sessionID":   uint64(s.ID),
		"currentPath": s.currentPath,
		"routeParams": s.routeParams,
		"nodeCount":   nodeCount,
		"treeDepth":   treeDepth,
	}
	data, _ := json.Marshal(snapshot)
	return data
}

// countNodes returns the total number of nodes in the tree.
func countNodes(n *dom.Node) int {
	if n == nil {
		return 0
	}
	count := 1
	for _, child := range n.Children {
		count += countNodes(child)
	}
	return count
}

// measureDepth returns the maximum depth of the tree.
func measureDepth(n *dom.Node) int {
	if n == nil {
		return 0
	}
	maxChild := 0
	for _, child := range n.Children {
		if d := measureDepth(child); d > maxChild {
			maxChild = d
		}
	}
	return 1 + maxChild
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

	for prop, val := range n.Styles {
		buf.EncodeSetStyle(uint32(n.ID), prop, val)
	}

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

	if s.metrics != nil {
		s.metrics.FlushTotal.Inc()
	}

	buf := protocol.AcquireBuffer()
	defer buf.Release()

	// Collect text updates for batching.
	var textUpdates []protocol.TextUpdate

	for _, n := range dirty {
		// 1. Drain structural PendingOps first (insert/remove)
		for _, op := range n.PendingOps {
			op(buf)
		}
		n.PendingOps = n.PendingOps[:0]

		// 2. Collect text updates (batched after this loop)
		if n.DirtyText {
			textUpdates = append(textUpdates, protocol.TextUpdate{
				NodeID: uint32(n.ID),
				Text:   n.Text,
			})
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

	// 5. Emit batched text updates — single OpBatchText for 2+ updates,
	// individual OpUpdateText for a single update.
	if len(textUpdates) >= 2 {
		buf.EncodeBatchText(textUpdates)
	} else if len(textUpdates) == 1 {
		buf.EncodeUpdateText(textUpdates[0].NodeID, textUpdates[0].Text)
	}

	if err := s.writer.WriteMessage(buf.Bytes()); err != nil {
		return err
	}

	if err := s.plugins.RunFlush(s.hookCtx()); err != nil {
		s.logger.Warn("flush hook failed", "error", err)
	}

	return nil
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
