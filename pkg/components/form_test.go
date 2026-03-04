package components

import (
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

func TestTextInput(t *testing.T) {
	value := dom.NewSignal("")
	node := TextInput("Enter name", value)

	if node.Tag != "input" {
		t.Fatalf("expected tag 'input', got %q", node.Tag)
	}
	if node.Attrs["type"] != "text" {
		t.Fatalf("expected type 'text', got %q", node.Attrs["type"])
	}
	if node.Attrs["placeholder"] != "Enter name" {
		t.Fatalf("expected placeholder 'Enter name', got %q", node.Attrs["placeholder"])
	}
	if node.Attrs["class"] == "" {
		t.Fatal("expected class attribute to be set")
	}
}

func TestCheckbox(t *testing.T) {
	checked := dom.NewSignal(false)
	node := Checkbox("Accept terms", checked)

	if node.Tag != "label" {
		t.Fatalf("expected tag 'label', got %q", node.Tag)
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected 2 children (input + span), got %d", len(node.Children))
	}
	input := node.Children[0]
	if input.Tag != "input" {
		t.Fatalf("expected first child tag 'input', got %q", input.Tag)
	}
	if input.Attrs["type"] != "checkbox" {
		t.Fatalf("expected type 'checkbox', got %q", input.Attrs["type"])
	}
	span := node.Children[1]
	if span.Tag != "span" {
		t.Fatalf("expected second child tag 'span', got %q", span.Tag)
	}
	if len(span.Children) != 1 || span.Children[0].Text != "Accept terms" {
		t.Fatal("expected span to contain label text 'Accept terms'")
	}
}

func TestSelect(t *testing.T) {
	selected := dom.NewSignal("b")
	node := Select([]string{"a", "b", "c"}, selected)

	if node.Tag != "select" {
		t.Fatalf("expected tag 'select', got %q", node.Tag)
	}
	if len(node.Children) != 3 {
		t.Fatalf("expected 3 option children, got %d", len(node.Children))
	}
	for i, opt := range node.Children {
		if opt.Tag != "option" {
			t.Fatalf("child %d: expected tag 'option', got %q", i, opt.Tag)
		}
	}
	if node.Children[1].Attrs["value"] != "b" {
		t.Fatalf("expected second option value 'b', got %q", node.Children[1].Attrs["value"])
	}
}

func TestTextArea(t *testing.T) {
	value := dom.NewSignal("hello")
	node := TextArea("Write here", value)

	if node.Tag != "textarea" {
		t.Fatalf("expected tag 'textarea', got %q", node.Tag)
	}
	if node.Attrs["placeholder"] != "Write here" {
		t.Fatalf("expected placeholder 'Write here', got %q", node.Attrs["placeholder"])
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 text child, got %d", len(node.Children))
	}
	if node.Children[0].Text != "hello" {
		t.Fatalf("expected text content 'hello', got %q", node.Children[0].Text)
	}
}
