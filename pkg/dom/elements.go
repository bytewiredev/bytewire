package dom

import "github.com/cbsframework/cbs/pkg/protocol"

// Option configures a Node during creation.
type Option func(*Node)

// Attr sets an HTML attribute.
func Attr(key, value string) Option {
	return func(n *Node) {
		n.Attrs[key] = value
	}
}

// ID sets the id attribute.
func ID(id string) Option {
	return Attr("id", id)
}

// Class sets the class attribute.
func Class(cls string) Option {
	return Attr("class", cls)
}

// Style sets an inline CSS property.
func Style(property, value string) Option {
	return func(n *Node) {
		n.Styles[property] = value
	}
}

// OnClick registers a click handler.
func OnClick(fn func([]byte)) Option {
	return func(n *Node) {
		n.Handlers[protocol.EventClick] = fn
	}
}

// OnInput registers an input handler.
func OnInput(fn func([]byte)) Option {
	return func(n *Node) {
		n.Handlers[protocol.EventInput] = fn
	}
}

// OnSubmit registers a submit handler.
func OnSubmit(fn func([]byte)) Option {
	return func(n *Node) {
		n.Handlers[protocol.EventSubmit] = fn
	}
}

// On registers a handler for any event type.
func On(eventType byte, fn func([]byte)) Option {
	return func(n *Node) {
		n.Handlers[eventType] = fn
	}
}

// Children appends child nodes.
func Children(children ...*Node) Option {
	return func(n *Node) {
		for _, c := range children {
			n.AppendChild(c)
		}
	}
}

// element creates a Node with the given tag and applies options.
func element(tag string, opts ...Option) *Node {
	n := newElement(tag)
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// --- HTML Element Constructors ---

func Div(opts ...Option) *Node    { return element("div", opts...) }
func Span(opts ...Option) *Node   { return element("span", opts...) }
func P(opts ...Option) *Node      { return element("p", opts...) }
func H1(opts ...Option) *Node     { return element("h1", opts...) }
func H2(opts ...Option) *Node     { return element("h2", opts...) }
func H3(opts ...Option) *Node     { return element("h3", opts...) }
func Button(opts ...Option) *Node { return element("button", opts...) }
func Input(opts ...Option) *Node  { return element("input", opts...) }
func Form(opts ...Option) *Node   { return element("form", opts...) }
func A(opts ...Option) *Node      { return element("a", opts...) }
func Ul(opts ...Option) *Node     { return element("ul", opts...) }
func Li(opts ...Option) *Node     { return element("li", opts...) }
func Nav(opts ...Option) *Node    { return element("nav", opts...) }
func Header(opts ...Option) *Node { return element("header", opts...) }
func Footer(opts ...Option) *Node { return element("footer", opts...) }
func Main(opts ...Option) *Node   { return element("main", opts...) }
func Section(opts ...Option) *Node { return element("section", opts...) }
func Article(opts ...Option) *Node { return element("article", opts...) }
func Img(opts ...Option) *Node    { return element("img", opts...) }
func Label(opts ...Option) *Node  { return element("label", opts...) }
func Table(opts ...Option) *Node  { return element("table", opts...) }
func Tr(opts ...Option) *Node     { return element("tr", opts...) }
func Td(opts ...Option) *Node     { return element("td", opts...) }
func Th(opts ...Option) *Node     { return element("th", opts...) }

// Text creates a text node.
func Text(content string) *Node {
	return newText(content)
}

// TextF creates a text node bound to a Signal. When the signal value changes,
// the node is automatically updated via binary delta.
func TextF[T comparable](s *Signal[T], format func(T) string) *Node {
	n := newText(format(s.Get()))
	n.SignalBound = true
	s.Observe(func(v T) {
		n.Text = format(v)
		n.Dirty = true
	})
	return n
}

// El creates a custom element with any tag name.
func El(tag string, opts ...Option) *Node {
	return element(tag, opts...)
}
