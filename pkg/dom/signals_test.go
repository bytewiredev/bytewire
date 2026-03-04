package dom

import (
	"sync"
	"testing"
)

func TestSignalGetSet(t *testing.T) {
	s := NewSignal(0)
	if s.Get() != 0 {
		t.Fatalf("expected 0, got %d", s.Get())
	}
	s.Set(42)
	if s.Get() != 42 {
		t.Fatalf("expected 42, got %d", s.Get())
	}
}

func TestSignalDirtyFlag(t *testing.T) {
	s := NewSignal("hello")
	if s.IsDirty() {
		t.Fatal("should not be dirty initially")
	}
	s.Set("world")
	if !s.IsDirty() {
		t.Fatal("should be dirty after set")
	}
	s.Flush()
	if s.IsDirty() {
		t.Fatal("should not be dirty after flush")
	}
}

func TestSignalNoOpSet(t *testing.T) {
	s := NewSignal(10)
	s.Set(10) // same value
	if s.IsDirty() {
		t.Fatal("should not be dirty when setting same value")
	}
}

func TestSignalObserve(t *testing.T) {
	s := NewSignal(0)
	var received int
	s.Observe(func(v int) { received = v })
	s.Set(5)
	if received != 5 {
		t.Fatalf("expected observer to receive 5, got %d", received)
	}
}

func TestSignalUnsubscribe(t *testing.T) {
	s := NewSignal(0)
	var count int
	unsub := s.Observe(func(_ int) { count++ })
	s.Set(1)
	unsub()
	s.Set(2)
	if count != 1 {
		t.Fatalf("expected 1 notification, got %d", count)
	}
}

func TestSignalUpdate(t *testing.T) {
	s := NewSignal(10)
	s.Update(func(v int) int { return v + 5 })
	if s.Get() != 15 {
		t.Fatalf("expected 15, got %d", s.Get())
	}
}

func TestComputed(t *testing.T) {
	count := NewSignal(3)
	doubled := Computed(count, func(v int) int { return v * 2 })

	if doubled.Get() != 6 {
		t.Fatalf("expected 6, got %d", doubled.Get())
	}
	count.Set(10)
	if doubled.Get() != 20 {
		t.Fatalf("expected 20, got %d", doubled.Get())
	}
}

func TestSignalConcurrency(t *testing.T) {
	s := NewSignal(0)
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			s.Set(v)
			_ = s.Get()
		}(i)
	}
	wg.Wait()
}

func BenchmarkSignalSet(b *testing.B) {
	s := NewSignal(0)
	b.ReportAllocs()
	for i := range b.N {
		s.Set(i)
	}
}
