package dom

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

// voidElements are HTML elements that cannot have children and are self-closing.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "source": true, "track": true, "wbr": true,
}

// RenderHTML renders a Node tree to an HTML string for server-side rendering.
// Element nodes include a data-bw-id attribute for hydration.
func RenderHTML(n *Node) string {
	if n == nil {
		return ""
	}

	if n.Type == TextNode {
		return html.EscapeString(n.Text)
	}

	var b strings.Builder

	b.WriteString("<")
	b.WriteString(n.Tag)

	// data-bw-id for hydration
	fmt.Fprintf(&b, ` data-bw-id="%d"`, n.ID)

	// Attributes in sorted order for determinism
	if len(n.Attrs) > 0 {
		keys := make([]string, 0, len(n.Attrs))
		for k := range n.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, ` %s="%s"`, html.EscapeString(k), html.EscapeString(n.Attrs[k]))
		}
	}

	// Styles
	if len(n.Styles) > 0 {
		keys := make([]string, 0, len(n.Styles))
		for k := range n.Styles {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString(` style="`)
		for i, k := range keys {
			if i > 0 {
				b.WriteString(";")
			}
			fmt.Fprintf(&b, "%s:%s", k, n.Styles[k])
		}
		b.WriteString(`"`)
	}

	// Void elements: self-closing, no children
	if voidElements[n.Tag] {
		b.WriteString(">")
		return b.String()
	}

	b.WriteString(">")

	// Children
	for _, child := range n.Children {
		b.WriteString(RenderHTML(child))
	}

	b.WriteString("</")
	b.WriteString(n.Tag)
	b.WriteString(">")

	return b.String()
}
