// Package router provides client-side routing for CBS applications.
// A Router is a Component that swaps child subtrees based on the current URL path.
package router

import (
	"strings"

	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/protocol"
)

// Route maps a URL pattern to a component.
type Route struct {
	pattern  string
	comp     engine.Component
	segments []string
}

// Router manages URL-based component switching.
type Router struct {
	routes       []Route
	notFound     engine.Component
	container    *dom.Node
	currentPath  string
	currentChild *dom.Node
	session      *engine.Session
}

// New creates a new Router.
func New() *Router {
	return &Router{}
}

// Handle registers a route pattern and its component.
// Patterns support exact matches and :param segments (e.g. "/users/:id").
func (r *Router) Handle(pattern string, comp engine.Component) *Router {
	r.routes = append(r.routes, Route{
		pattern:  pattern,
		comp:     comp,
		segments: splitPath(pattern),
	})
	return r
}

// NotFound sets the component rendered for unmatched routes.
func (r *Router) NotFound(comp engine.Component) *Router {
	r.notFound = comp
	return r
}

// Mount returns the router's root node and registers navigation handling.
// This is the Component function passed to engine.
func (r *Router) Mount(s *engine.Session) *dom.Node {
	r.session = s
	r.container = dom.Div()

	// Determine initial path
	path := s.CurrentPath()
	if path == "" {
		path = "/"
	}

	r.renderPath(path)
	s.SetNavHandler(r.navigate)

	return r.container
}

// navigate handles a client navigation event.
func (r *Router) navigate(path string) {
	if path == r.currentPath {
		return
	}
	r.swapRoute(path)
}

// renderPath renders the initial route without emitting remove ops.
func (r *Router) renderPath(path string) {
	comp, params := r.match(path)
	r.currentPath = path
	r.session.SetCurrentPath(path)
	r.session.SetRouteParams(params)

	if comp != nil {
		child := comp(r.session)
		r.container.AppendChild(child)
		r.currentChild = child
	}
}

// swapRoute removes the current child and renders the new route.
func (r *Router) swapRoute(path string) {
	comp, params := r.match(path)
	r.currentPath = path
	r.session.SetCurrentPath(path)
	r.session.SetRouteParams(params)

	// Remove old child
	if r.currentChild != nil {
		oldChild := r.currentChild
		r.container.PendingOps = append(r.container.PendingOps, func(buf *protocol.Buffer) {
			buf.EncodeRemoveNode(uint32(oldChild.ID))
		})
		r.container.RemoveChild(oldChild)
		r.currentChild = nil
	}

	// Render new child
	if comp != nil {
		child := comp(r.session)
		r.container.AppendChild(child)
		r.currentChild = child
		dom.QueueInsert(r.container, child)
	}

	// Push history so the browser URL updates
	r.container.PendingOps = append(r.container.PendingOps, func(buf *protocol.Buffer) {
		buf.EncodePushHistory(path)
	})

	r.container.Dirty = true
}

// match finds the first matching route for the given path.
func (r *Router) match(path string) (engine.Component, map[string]string) {
	pathSegs := splitPath(path)
	for _, route := range r.routes {
		if params, ok := matchSegments(route.segments, pathSegs); ok {
			return route.comp, params
		}
	}
	if r.notFound != nil {
		return r.notFound, nil
	}
	return nil, nil
}

// splitPath splits a URL path into segments, filtering empty strings.
func splitPath(path string) []string {
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// matchSegments compares pattern segments against path segments.
// Segments starting with ":" are treated as named parameters.
func matchSegments(pattern, path []string) (map[string]string, bool) {
	if len(pattern) != len(path) {
		return nil, false
	}
	params := make(map[string]string)
	for i, seg := range pattern {
		if strings.HasPrefix(seg, ":") {
			params[seg[1:]] = path[i]
			continue
		}
		if seg != path[i] {
			return nil, false
		}
	}
	return params, true
}
