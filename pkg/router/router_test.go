package router

import (
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
)

func dummyComp(text string) engine.Component {
	return func(s *engine.Session) *dom.Node {
		return dom.Div(dom.Children(dom.Text(text)))
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"/", 0},
		{"/about", 1},
		{"/users/123", 2},
		{"/a/b/c", 3},
		{"", 0},
	}
	for _, tt := range tests {
		got := splitPath(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitPath(%q) = %d segments, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestMatchSegmentsExact(t *testing.T) {
	pattern := splitPath("/about")
	path := splitPath("/about")
	params, ok := matchSegments(pattern, path)
	if !ok {
		t.Fatal("expected match")
	}
	if len(params) != 0 {
		t.Fatalf("expected no params, got %v", params)
	}
}

func TestMatchSegmentsParam(t *testing.T) {
	pattern := splitPath("/users/:id")
	path := splitPath("/users/42")
	params, ok := matchSegments(pattern, path)
	if !ok {
		t.Fatal("expected match")
	}
	if params["id"] != "42" {
		t.Fatalf("expected id=42, got %q", params["id"])
	}
}

func TestMatchSegmentsMultipleParams(t *testing.T) {
	pattern := splitPath("/org/:orgID/users/:userID")
	path := splitPath("/org/acme/users/99")
	params, ok := matchSegments(pattern, path)
	if !ok {
		t.Fatal("expected match")
	}
	if params["orgID"] != "acme" || params["userID"] != "99" {
		t.Fatalf("unexpected params: %v", params)
	}
}

func TestMatchSegmentsNoMatch(t *testing.T) {
	pattern := splitPath("/about")
	path := splitPath("/contact")
	_, ok := matchSegments(pattern, path)
	if ok {
		t.Fatal("expected no match")
	}
}

func TestMatchSegmentsLengthMismatch(t *testing.T) {
	pattern := splitPath("/users/:id")
	path := splitPath("/users")
	_, ok := matchSegments(pattern, path)
	if ok {
		t.Fatal("expected no match for length mismatch")
	}
}

func TestRouterHandle(t *testing.T) {
	r := New()
	r.Handle("/", dummyComp("home"))
	r.Handle("/about", dummyComp("about"))

	if len(r.routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(r.routes))
	}
}

func TestRouterMatch(t *testing.T) {
	r := New()
	r.Handle("/", dummyComp("home"))
	r.Handle("/about", dummyComp("about"))
	r.Handle("/users/:id", dummyComp("user"))
	r.NotFound(dummyComp("404"))

	comp, params := r.match("/")
	if comp == nil {
		t.Fatal("expected match for /")
	}
	if len(params) != 0 {
		t.Fatalf("expected no params for /, got %v", params)
	}

	comp, params = r.match("/users/42")
	if comp == nil {
		t.Fatal("expected match for /users/42")
	}
	if params["id"] != "42" {
		t.Fatalf("expected id=42, got %q", params["id"])
	}

	comp, _ = r.match("/nonexistent")
	if comp == nil {
		t.Fatal("expected notFound component")
	}
}

func TestRouterMatchNotFoundNil(t *testing.T) {
	r := New()
	r.Handle("/about", dummyComp("about"))

	comp, _ := r.match("/nonexistent")
	if comp != nil {
		t.Fatal("expected nil when no notFound set")
	}
}

func TestSplitQueryNoQuery(t *testing.T) {
	path, query := splitQuery("/about")
	if path != "/about" {
		t.Fatalf("expected /about, got %q", path)
	}
	if query != nil {
		t.Fatalf("expected nil query, got %v", query)
	}
}

func TestSplitQuerySingle(t *testing.T) {
	path, query := splitQuery("/search?q=hello")
	if path != "/search" {
		t.Fatalf("expected /search, got %q", path)
	}
	if query["q"] != "hello" {
		t.Fatalf("expected q=hello, got %q", query["q"])
	}
}

func TestSplitQueryMultiple(t *testing.T) {
	path, query := splitQuery("/search?q=hello&page=2&sort=asc")
	if path != "/search" {
		t.Fatalf("expected /search, got %q", path)
	}
	if query["q"] != "hello" || query["page"] != "2" || query["sort"] != "asc" {
		t.Fatalf("unexpected query: %v", query)
	}
}

func TestSplitQueryEmpty(t *testing.T) {
	path, query := splitQuery("/search?")
	if path != "/search" {
		t.Fatalf("expected /search, got %q", path)
	}
	if query != nil {
		t.Fatalf("expected nil query for empty query string, got %v", query)
	}
}

func TestSplitQueryKeyOnly(t *testing.T) {
	path, query := splitQuery("/page?debug")
	if path != "/page" {
		t.Fatalf("expected /page, got %q", path)
	}
	if query["debug"] != "" {
		t.Fatalf("expected debug='', got %q", query["debug"])
	}
	if _, ok := query["debug"]; !ok {
		t.Fatal("expected debug key to exist")
	}
}

func TestRouterMatchWithQueryString(t *testing.T) {
	r := New()
	r.Handle("/search", dummyComp("search"))

	// match should work on the path without query string
	comp, _ := r.match("/search")
	if comp == nil {
		t.Fatal("expected match for /search")
	}
}
