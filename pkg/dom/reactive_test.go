package dom

import (
	"testing"

	"github.com/bytewiredev/bytewire/pkg/protocol"
)

func TestListSignalSetAndGet(t *testing.T) {
	s := NewListSignal([]string{"a", "b"})
	got := s.Get()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("expected [a b], got %v", got)
	}
	// Verify defensive copy — mutating returned slice doesn't affect signal.
	got[0] = "z"
	again := s.Get()
	if again[0] != "a" {
		t.Fatal("Get returned a reference, not a copy")
	}
}

func TestListSignalObserve(t *testing.T) {
	s := NewListSignal([]int{1, 2})
	var observed []int
	s.Observe(func(items []int) {
		observed = items
	})
	s.Set([]int{3, 4, 5})
	if len(observed) != 3 || observed[0] != 3 {
		t.Fatalf("observer not called correctly: %v", observed)
	}
}

func TestListSignalAppend(t *testing.T) {
	s := NewListSignal([]string{"x"})
	var observed []string
	s.Observe(func(items []string) {
		observed = items
	})
	s.Append("y")
	if len(observed) != 2 || observed[1] != "y" {
		t.Fatalf("expected [x y], got %v", observed)
	}
}

func TestListSignalUpdate(t *testing.T) {
	s := NewListSignal([]int{1, 2, 3})
	s.Update(func(items []int) []int {
		return items[:2] // remove last
	})
	got := s.Get()
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
}

func TestIfSwitchesBranches(t *testing.T) {
	s := NewSignal(true)
	container := If(s, func(v bool) bool { return v },
		func() *Node { return Div(Children(Text("yes"))) },
		func() *Node { return Div(Children(Text("no"))) },
	)

	if len(container.Children) != 1 || container.Children[0].Children[0].Text != "yes" {
		t.Fatal("expected 'yes' branch initially")
	}

	s.Set(false)
	if len(container.PendingOps) == 0 {
		t.Fatal("expected PendingOps after condition flip")
	}
	if len(container.Children) != 1 || container.Children[0].Children[0].Text != "no" {
		t.Fatal("expected 'no' branch after flip")
	}
}

func TestIfNoOpOnSameCondition(t *testing.T) {
	s := NewSignal(5)
	container := If(s, func(v int) bool { return v > 0 },
		func() *Node { return Text("positive") },
		nil,
	)
	s.Set(10) // still > 0
	if len(container.PendingOps) != 0 {
		t.Fatal("should not have PendingOps when condition unchanged")
	}
}

func TestIfNilElse(t *testing.T) {
	s := NewSignal(false)
	container := If(s, func(v bool) bool { return v },
		func() *Node { return Text("shown") },
		nil,
	)
	if len(container.Children) != 0 {
		t.Fatal("expected empty container when false with nil else")
	}
	s.Set(true)
	if len(container.Children) != 1 {
		t.Fatal("expected one child after flip to true")
	}
}

func TestForAppendItem(t *testing.T) {
	s := NewListSignal([]string{"a"})
	container := For(s, func(item string) string { return item },
		func(item string) *Node { return Li(Children(Text(item))) },
	)
	if len(container.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(container.Children))
	}
	s.Set([]string{"a", "b"})
	if len(container.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(container.Children))
	}
	if len(container.PendingOps) == 0 {
		t.Fatal("expected PendingOps for new item")
	}
}

func TestForRemoveItem(t *testing.T) {
	s := NewListSignal([]string{"a", "b", "c"})
	container := For(s, func(item string) string { return item },
		func(item string) *Node { return Li(Children(Text(item))) },
	)
	s.Set([]string{"a", "c"})
	if len(container.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(container.Children))
	}
	// Verify PendingOps includes remove
	buf := protocol.AcquireBuffer()
	defer buf.Release()
	for _, op := range container.PendingOps {
		op(buf)
	}
	if buf.Len() == 0 {
		t.Fatal("expected opcodes for removal")
	}
}

func TestForReorderItems(t *testing.T) {
	s := NewListSignal([]string{"a", "b", "c"})
	container := For(s, func(item string) string { return item },
		func(item string) *Node { return Li(Children(Text(item))) },
	)
	s.Set([]string{"c", "a", "b"})
	if len(container.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(container.Children))
	}
	// First child should be the "c" node
	if container.Children[0].Children[0].Text != "c" {
		t.Fatal("expected reordered children")
	}
}

func TestAttrFDirtyTracking(t *testing.T) {
	s := NewSignal("active")
	n := Div(AttrF(s, "class", func(v string) string { return v }))
	if n.Attrs["class"] != "active" {
		t.Fatal("initial attr not set")
	}
	s.Set("inactive")
	if n.DirtyAttrs["class"] != "inactive" {
		t.Fatal("DirtyAttrs not updated")
	}
	if !n.Dirty {
		t.Fatal("node not marked dirty")
	}
}

func TestAttrFEmptyRemoves(t *testing.T) {
	s := NewSignal("visible")
	n := Div(AttrF(s, "hidden", func(v string) string {
		if v == "hidden" {
			return "true"
		}
		return ""
	}))
	if n.Attrs["hidden"] != "" {
		t.Fatal("expected empty initial attr for visible")
	}
	s.Set("hidden")
	if n.DirtyAttrs["hidden"] != "true" {
		t.Fatalf("expected 'true', got '%s'", n.DirtyAttrs["hidden"])
	}
	s.Set("visible")
	if n.DirtyAttrs["hidden"] != "" {
		t.Fatal("expected empty string for removal")
	}
}

func TestStyleFDirtyTracking(t *testing.T) {
	s := NewSignal("red")
	n := Div(StyleF(s, "color", func(v string) string { return v }))
	if n.Styles["color"] != "red" {
		t.Fatal("initial style not set")
	}
	s.Set("blue")
	if n.DirtyStyles["color"] != "blue" {
		t.Fatal("DirtyStyles not updated")
	}
}

func TestFlushDrainsAllDirtyTypes(t *testing.T) {
	// Create a node with all dirty types
	n := Div()
	n.Dirty = true
	n.DirtyText = true
	n.Text = "updated"
	n.DirtyAttrs = map[string]string{"class": "new"}
	n.DirtyStyles = map[string]string{"color": "red"}
	n.PendingOps = append(n.PendingOps, func(buf *protocol.Buffer) {
		buf.EncodeRemoveNode(999)
	})

	buf := protocol.AcquireBuffer()
	defer buf.Release()

	// Drain PendingOps
	for _, op := range n.PendingOps {
		op(buf)
	}
	n.PendingOps = n.PendingOps[:0]

	// Emit text
	if n.DirtyText {
		buf.EncodeUpdateText(uint32(n.ID), n.Text)
		n.DirtyText = false
	}

	// Emit attrs
	for key, val := range n.DirtyAttrs {
		if val == "" {
			buf.EncodeRemoveAttr(uint32(n.ID), key)
		} else {
			buf.EncodeSetAttr(uint32(n.ID), key, val)
		}
	}
	clear(n.DirtyAttrs)

	// Emit styles
	for prop, val := range n.DirtyStyles {
		buf.EncodeSetStyle(uint32(n.ID), prop, val)
	}
	clear(n.DirtyStyles)

	n.Dirty = false

	// Verify everything was emitted
	msgs, err := protocol.DecodeAll(buf.Bytes())
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(msgs) != 4 { // remove + text + attr + style
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if msgs[0].Op != protocol.OpRemoveNode {
		t.Fatal("expected OpRemoveNode first")
	}
	if msgs[1].Op != protocol.OpUpdateText {
		t.Fatal("expected OpUpdateText second")
	}
	if msgs[2].Op != protocol.OpSetAttr {
		t.Fatal("expected OpSetAttr third")
	}
	if msgs[3].Op != protocol.OpSetStyle {
		t.Fatal("expected OpSetStyle fourth")
	}
}
