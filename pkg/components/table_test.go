package components

import (
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

type testItem struct {
	ID   string
	Name string
}

func TestTable(t *testing.T) {
	rows := dom.NewListSignal([]testItem{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
	})

	cols := []Column[testItem]{
		{Header: "ID", Render: func(item testItem) *dom.Node { return dom.Text(item.ID) }},
		{Header: "Name", Render: func(item testItem) *dom.Node { return dom.Text(item.Name) }},
	}

	node := Table(rows, func(item testItem) string { return item.ID }, cols)

	if node.Tag != "table" {
		t.Fatalf("expected tag 'table', got %q", node.Tag)
	}
	if len(node.Children) != 2 {
		t.Fatalf("expected 2 children (thead + For container), got %d", len(node.Children))
	}

	thead := node.Children[0]
	if thead.Tag != "thead" {
		t.Fatalf("expected first child tag 'thead', got %q", thead.Tag)
	}
	// thead -> tr -> th*2
	if len(thead.Children) != 1 {
		t.Fatalf("expected thead to have 1 tr, got %d", len(thead.Children))
	}
	headerRow := thead.Children[0]
	if len(headerRow.Children) != 2 {
		t.Fatalf("expected 2 th cells, got %d", len(headerRow.Children))
	}
	if headerRow.Children[0].Tag != "th" {
		t.Fatalf("expected th tag, got %q", headerRow.Children[0].Tag)
	}

	// For container is a div with 2 tr children (one per row)
	forContainer := node.Children[1]
	if forContainer.Tag != "div" {
		t.Fatalf("expected For container tag 'div', got %q", forContainer.Tag)
	}
	if len(forContainer.Children) != 2 {
		t.Fatalf("expected 2 data rows, got %d", len(forContainer.Children))
	}
	firstRow := forContainer.Children[0]
	if firstRow.Tag != "tr" {
		t.Fatalf("expected row tag 'tr', got %q", firstRow.Tag)
	}
	if len(firstRow.Children) != 2 {
		t.Fatalf("expected 2 td cells per row, got %d", len(firstRow.Children))
	}
}
