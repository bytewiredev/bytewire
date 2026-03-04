package components

import (
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

func TestModalVisible(t *testing.T) {
	visible := dom.NewSignal(true)
	node := Modal("Test Modal", visible, dom.Text("body content"))

	// Modal uses dom.If which returns a container div
	if node.Tag != "div" {
		t.Fatalf("expected tag 'div' (If container), got %q", node.Tag)
	}
	// When visible=true, the container should have 1 child (the backdrop)
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child (backdrop) when visible, got %d", len(node.Children))
	}

	backdrop := node.Children[0]
	if backdrop.Tag != "div" {
		t.Fatalf("expected backdrop tag 'div', got %q", backdrop.Tag)
	}

	// backdrop -> dialog
	if len(backdrop.Children) != 1 {
		t.Fatalf("expected 1 child (dialog) in backdrop, got %d", len(backdrop.Children))
	}
	dialog := backdrop.Children[0]
	if dialog.Tag != "div" {
		t.Fatalf("expected dialog tag 'div', got %q", dialog.Tag)
	}

	// dialog -> header + body
	if len(dialog.Children) != 2 {
		t.Fatalf("expected 2 children (header + body) in dialog, got %d", len(dialog.Children))
	}
}

func TestModalHidden(t *testing.T) {
	visible := dom.NewSignal(false)
	node := Modal("Hidden Modal", visible, dom.Text("hidden"))

	if node.Tag != "div" {
		t.Fatalf("expected tag 'div' (If container), got %q", node.Tag)
	}
	// When visible=false, container should have no children
	if len(node.Children) != 0 {
		t.Fatalf("expected 0 children when hidden, got %d", len(node.Children))
	}
}
