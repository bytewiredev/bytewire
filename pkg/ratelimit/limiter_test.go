package ratelimit

import (
	"testing"
	"time"
)

func TestAllowBurst(t *testing.T) {
	l := New(10, 5)

	// Should allow exactly burst count.
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("Allow() returned false on call %d, expected true", i+1)
		}
	}

	// Next call should be denied.
	if l.Allow() {
		t.Fatal("Allow() returned true after burst exhausted, expected false")
	}
}

func TestRefill(t *testing.T) {
	l := New(100, 5)

	// Exhaust all tokens.
	for i := 0; i < 5; i++ {
		l.Allow()
	}
	if l.Allow() {
		t.Fatal("Allow() should be false after exhaustion")
	}

	// Simulate time passing to refill tokens.
	l.mu.Lock()
	l.lastTime = l.lastTime.Add(-100 * time.Millisecond) // 100ms at 100/s = 10 tokens, capped at 5
	l.mu.Unlock()

	// Should have refilled up to burst (5).
	for i := 0; i < 5; i++ {
		if !l.Allow() {
			t.Fatalf("Allow() returned false on refill call %d", i+1)
		}
	}
	if l.Allow() {
		t.Fatal("Allow() should be false after consuming refilled burst")
	}
}

func TestPartialRefill(t *testing.T) {
	l := New(10, 5)

	// Exhaust all tokens.
	for i := 0; i < 5; i++ {
		l.Allow()
	}

	// Simulate 150ms elapsed: 10 tokens/s * 0.15s = 1.5 tokens -> 1 allowed.
	l.mu.Lock()
	l.lastTime = l.lastTime.Add(-150 * time.Millisecond)
	l.mu.Unlock()

	if !l.Allow() {
		t.Fatal("Allow() should succeed after partial refill")
	}
	if l.Allow() {
		t.Fatal("Allow() should fail, only ~0.5 tokens remaining")
	}
}

func TestZeroBurst(t *testing.T) {
	l := New(10, 0)
	if l.Allow() {
		t.Fatal("Allow() should be false with zero burst")
	}
}
