// Package dom provides the developer-facing API for building Bytewire UIs in pure Go.
package dom

import (
	"sync"
	"sync/atomic"
)

// SignalID is a unique identifier for a reactive signal.
type SignalID uint64

var signalCounter atomic.Uint64

func nextSignalID() SignalID {
	return SignalID(signalCounter.Add(1))
}

// Observable is the read/subscribe interface shared by Signal and ListSignal.
type Observable[T any] interface {
	Get() T
	Observe(fn func(T)) func()
}

// Signal is a reactive state container. When its value changes,
// any bound DOM nodes are automatically flagged for binary delta emission.
type Signal[T comparable] struct {
	id        SignalID
	mu        sync.RWMutex
	value     T
	dirty     atomic.Bool
	observers []func(T)
}

// NewSignal creates a Signal with an initial value.
func NewSignal[T comparable](initial T) *Signal[T] {
	return &Signal[T]{
		id:    nextSignalID(),
		value: initial,
	}
}

// Get returns the current value.
func (s *Signal[T]) Get() T {
	s.mu.RLock()
	v := s.value
	s.mu.RUnlock()
	return v
}

// Set updates the value. If the new value differs from the current,
// the signal is marked dirty and all observers are notified.
func (s *Signal[T]) Set(v T) {
	s.mu.Lock()
	if s.value == v {
		s.mu.Unlock()
		return
	}
	s.value = v
	s.dirty.Store(true)
	observers := make([]func(T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, fn := range observers {
		if fn != nil {
			fn(v)
		}
	}
}

// Update applies a transform function to the current value.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	old := s.value
	next := fn(old)
	if old == next {
		s.mu.Unlock()
		return
	}
	s.value = next
	s.dirty.Store(true)
	observers := make([]func(T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, fn := range observers {
		if fn != nil {
			fn(next)
		}
	}
}

// IsDirty returns true if the value has changed since last flush.
func (s *Signal[T]) IsDirty() bool {
	return s.dirty.Load()
}

// Flush clears the dirty flag. Called by the engine after emitting deltas.
func (s *Signal[T]) Flush() {
	s.dirty.Store(false)
}

// ID returns the signal's unique identifier.
func (s *Signal[T]) ID() SignalID {
	return s.id
}

// Observe registers a callback invoked whenever the value changes.
// Returns an unsubscribe function.
func (s *Signal[T]) Observe(fn func(T)) func() {
	s.mu.Lock()
	s.observers = append(s.observers, fn)
	idx := len(s.observers) - 1
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		// Nil out to avoid shifting; compacted on next Set if needed.
		if idx < len(s.observers) {
			s.observers[idx] = nil
		}
	}
}

// Computed creates a derived signal that recomputes when the source changes.
func Computed[T any, U comparable](source Observable[T], derive func(T) U) *Signal[U] {
	computed := NewSignal(derive(source.Get()))
	source.Observe(func(v T) {
		computed.Set(derive(v))
	})
	return computed
}

// BoxSignal is a reactive container for non-comparable values (e.g., structs
// containing slices). Unlike Signal[T], it always notifies on Set since the
// value cannot be compared with ==.
type BoxSignal[T any] struct {
	id        SignalID
	mu        sync.RWMutex
	value     T
	observers []func(T)
}

// NewBoxSignal creates a BoxSignal with an initial value.
func NewBoxSignal[T any](initial T) *BoxSignal[T] {
	return &BoxSignal[T]{
		id:    nextSignalID(),
		value: initial,
	}
}

// Get returns the current value.
func (s *BoxSignal[T]) Get() T {
	s.mu.RLock()
	v := s.value
	s.mu.RUnlock()
	return v
}

// Set updates the value and notifies all observers.
func (s *BoxSignal[T]) Set(v T) {
	s.mu.Lock()
	s.value = v
	observers := make([]func(T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, fn := range observers {
		if fn != nil {
			fn(v)
		}
	}
}

// Update applies a transform function to the current value.
func (s *BoxSignal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	s.value = fn(s.value)
	v := s.value
	observers := make([]func(T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	for _, fn := range observers {
		if fn != nil {
			fn(v)
		}
	}
}

// ID returns the signal's unique identifier.
func (s *BoxSignal[T]) ID() SignalID {
	return s.id
}

// Observe registers a callback invoked whenever the value changes.
// Returns an unsubscribe function.
func (s *BoxSignal[T]) Observe(fn func(T)) func() {
	s.mu.Lock()
	s.observers = append(s.observers, fn)
	idx := len(s.observers) - 1
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if idx < len(s.observers) {
			s.observers[idx] = nil
		}
	}
}

// ListSignal is a reactive container for slices. Unlike Signal[T], it does
// not require comparable elements and always notifies on Set (since slices
// cannot be compared with ==).
type ListSignal[T any] struct {
	id        SignalID
	mu        sync.RWMutex
	items     []T
	observers []func([]T)
}

// NewListSignal creates a ListSignal with an initial slice.
func NewListSignal[T any](initial []T) *ListSignal[T] {
	cp := make([]T, len(initial))
	copy(cp, initial)
	return &ListSignal[T]{
		id:    nextSignalID(),
		items: cp,
	}
}

// Get returns a defensive copy of the current items.
func (s *ListSignal[T]) Get() []T {
	s.mu.RLock()
	cp := make([]T, len(s.items))
	copy(cp, s.items)
	s.mu.RUnlock()
	return cp
}

// Set replaces the items and notifies all observers.
func (s *ListSignal[T]) Set(items []T) {
	s.mu.Lock()
	s.items = make([]T, len(items))
	copy(s.items, items)
	observers := make([]func([]T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	snapshot := s.Get()
	for _, fn := range observers {
		if fn != nil {
			fn(snapshot)
		}
	}
}

// Append adds an item and notifies observers.
func (s *ListSignal[T]) Append(item T) {
	s.mu.Lock()
	s.items = append(s.items, item)
	observers := make([]func([]T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	snapshot := s.Get()
	for _, fn := range observers {
		if fn != nil {
			fn(snapshot)
		}
	}
}

// Update applies a transform function to the current items.
func (s *ListSignal[T]) Update(fn func([]T) []T) {
	s.mu.Lock()
	s.items = fn(s.items)
	observers := make([]func([]T), len(s.observers))
	copy(observers, s.observers)
	s.mu.Unlock()

	snapshot := s.Get()
	for _, fn := range observers {
		if fn != nil {
			fn(snapshot)
		}
	}
}

// ID returns the signal's unique identifier.
func (s *ListSignal[T]) ID() SignalID {
	return s.id
}

// Observe registers a callback invoked whenever the items change.
// Returns an unsubscribe function.
func (s *ListSignal[T]) Observe(fn func([]T)) func() {
	s.mu.Lock()
	s.observers = append(s.observers, fn)
	idx := len(s.observers) - 1
	s.mu.Unlock()

	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if idx < len(s.observers) {
			s.observers[idx] = nil
		}
	}
}
