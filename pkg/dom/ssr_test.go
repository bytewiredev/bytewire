package dom

import (
	"fmt"
	"strings"
	"testing"
)

func TestRenderHTML(t *testing.T) {
	tests := []struct {
		name     string
		node     *Node
		contains []string // substrings the output must contain
		exact    string   // if non-empty, output must match exactly
	}{
		{
			name: "nil node returns empty string",
			node: nil,
			exact: "",
		},
		{
			name: "simple div with text child",
			node: func() *Node {
				d := Div()
				d.AppendChild(Text("Hello"))
				return d
			}(),
			contains: []string{"<div data-bw-id=", ">Hello</div>"},
		},
		{
			name: "nested elements div > span > text",
			node: func() *Node {
				d := Div(Children(
					Span(Children(Text("nested"))),
				))
				return d
			}(),
			contains: []string{"<div data-bw-id=", "<span data-bw-id=", ">nested</span>", "</div>"},
		},
		{
			name: "void element img with src attr",
			node: func() *Node {
				return Img(Attr("src", "test.png"))
			}(),
			contains: []string{"<img data-bw-id=", `src="test.png"`},
		},
		{
			name: "void element has no closing tag",
			node: Img(Attr("src", "x.png")),
			contains: []string{"<img data-bw-id="},
		},
		{
			name: "attributes rendered correctly",
			node: Div(Attr("class", "foo"), Attr("id", "bar")),
			contains: []string{`class="foo"`, `id="bar"`},
		},
		{
			name: "styles rendered correctly",
			node: Div(Style("color", "red"), Style("margin", "10px")),
			contains: []string{`style="`, "color:red", "margin:10px"},
		},
		{
			name: "text escaping prevents XSS",
			node: func() *Node {
				d := Div()
				d.AppendChild(Text("<script>alert('xss')</script>"))
				return d
			}(),
			contains: []string{"&lt;script&gt;"},
		},
		{
			name: "data-bw-id present on all element nodes",
			node: func() *Node {
				return Div(Children(
					Span(),
					P(),
				))
			}(),
			contains: []string{"data-bw-id="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderHTML(tt.node)

			if tt.exact != "" || (tt.node == nil) {
				if got != tt.exact {
					t.Fatalf("expected %q, got %q", tt.exact, got)
				}
				return
			}

			for _, sub := range tt.contains {
				if !strings.Contains(got, sub) {
					t.Errorf("expected output to contain %q, got:\n%s", sub, got)
				}
			}
		})
	}
}

func TestRenderHTML_DataBWIDOnAllElements(t *testing.T) {
	root := Div(Children(
		Span(Children(Text("a"))),
		P(Children(Text("b"))),
	))
	html := RenderHTML(root)

	// Count data-bw-id occurrences — should be 3 (div, span, p), not on text nodes
	count := strings.Count(html, "data-bw-id=")
	if count != 3 {
		t.Fatalf("expected 3 data-bw-id attributes, got %d in:\n%s", count, html)
	}
}

func TestRenderHTML_VoidElementNoClosingTag(t *testing.T) {
	img := Img(Attr("src", "test.png"))
	html := RenderHTML(img)

	if strings.Contains(html, "</img>") {
		t.Fatalf("void element should not have closing tag, got:\n%s", html)
	}
}

func TestRenderHTML_CorrectNodeIDs(t *testing.T) {
	d := Div()
	html := RenderHTML(d)

	expected := fmt.Sprintf(`data-bw-id="%d"`, d.ID)
	if !strings.Contains(html, expected) {
		t.Fatalf("expected %s in output, got:\n%s", expected, html)
	}
}
