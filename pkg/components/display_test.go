package components

import (
	"strings"
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

func TestCard(t *testing.T) {
	node := Card("My Card", dom.Text("content"))

	if node.Tag != "div" {
		t.Fatalf("expected tag 'div', got %q", node.Tag)
	}
	// header div + content text node
	if len(node.Children) != 2 {
		t.Fatalf("expected 2 children (header + content), got %d", len(node.Children))
	}
	header := node.Children[0]
	if header.Tag != "div" {
		t.Fatalf("expected header tag 'div', got %q", header.Tag)
	}
	// header -> h3 -> text
	h3 := header.Children[0]
	if h3.Tag != "h3" {
		t.Fatalf("expected h3 tag, got %q", h3.Tag)
	}
	if len(h3.Children) != 1 || h3.Children[0].Text != "My Card" {
		t.Fatal("expected h3 to contain 'My Card'")
	}
}

func TestBadge(t *testing.T) {
	tests := []struct {
		variant  string
		wantText string
	}{
		{"success", "OK"},
		{"error", "Fail"},
		{"warning", "Warn"},
		{"default", "Info"},
	}
	for _, tt := range tests {
		node := Badge(tt.wantText, tt.variant)
		if node.Tag != "span" {
			t.Fatalf("variant %s: expected tag 'span', got %q", tt.variant, node.Tag)
		}
		if len(node.Children) != 1 || node.Children[0].Text != tt.wantText {
			t.Fatalf("variant %s: expected text %q", tt.variant, tt.wantText)
		}
	}
}

func TestAlert(t *testing.T) {
	node := Alert("Something happened", "info", false)

	if node.Tag != "div" {
		t.Fatalf("expected tag 'div', got %q", node.Tag)
	}
	if !strings.Contains(node.Attrs["class"], "border-l-4") {
		t.Fatal("expected border-l-4 class on alert")
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child (content span), got %d", len(node.Children))
	}
	if node.Children[0].Tag != "span" {
		t.Fatalf("expected child tag 'span', got %q", node.Children[0].Tag)
	}
}

func TestAlertDismissible(t *testing.T) {
	node := Alert("Dismiss me", "error", true)

	// Dismissible alert uses dom.If -> container div
	if node.Tag != "div" {
		t.Fatalf("expected tag 'div' (If container), got %q", node.Tag)
	}
	// When initially visible, should have 1 child (the alert div)
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child when visible, got %d", len(node.Children))
	}
	alertDiv := node.Children[0]
	// alert div -> content span + close button
	if len(alertDiv.Children) != 2 {
		t.Fatalf("expected 2 children (content + close btn), got %d", len(alertDiv.Children))
	}
}

func TestSpinner(t *testing.T) {
	node := Spinner()

	if node.Tag != "div" {
		t.Fatalf("expected tag 'div', got %q", node.Tag)
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child (spinner circle), got %d", len(node.Children))
	}
	circle := node.Children[0]
	if circle.Tag != "div" {
		t.Fatalf("expected inner tag 'div', got %q", circle.Tag)
	}
	if !strings.Contains(circle.Attrs["class"], "animate-spin") {
		t.Fatal("expected animate-spin class on spinner")
	}
}
