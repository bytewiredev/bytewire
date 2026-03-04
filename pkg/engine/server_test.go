package engine

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/bytewiredev/bytewire/pkg/dom"
)

// mockWriter is a no-op writer for testing.
type mockWriter struct{}

func (m *mockWriter) WriteMessage(data []byte) error { return nil }
func (m *mockWriter) Close() error                   { return nil }

func newTestServer() *Server {
	return NewServer(":0", nil, func(s *Session) *dom.Node { return nil })
}

func newTestSession() *Session {
	return NewSession(context.Background(), &mockWriter{}, slog.Default(), nil, nil, nil)
}

func TestSessionRegistry_RegisterAndLookup(t *testing.T) {
	srv := newTestServer()

	sess := newTestSession()
	srv.mu.Lock()
	srv.sessions[sess.ID] = sess
	srv.mu.Unlock()

	got := srv.Session(sess.ID)
	if got == nil {
		t.Fatal("expected session, got nil")
	}
	if got.ID != sess.ID {
		t.Fatalf("expected session ID %d, got %d", sess.ID, got.ID)
	}
}

func TestSessionRegistry_LookupMissing(t *testing.T) {
	srv := newTestServer()

	got := srv.Session(SessionID(99999))
	if got != nil {
		t.Fatalf("expected nil for missing session, got %v", got)
	}
}

func TestSessionRegistry_Remove(t *testing.T) {
	srv := newTestServer()

	sess := newTestSession()
	srv.mu.Lock()
	srv.sessions[sess.ID] = sess
	srv.mu.Unlock()

	// Remove
	srv.mu.Lock()
	delete(srv.sessions, sess.ID)
	srv.mu.Unlock()

	got := srv.Session(sess.ID)
	if got != nil {
		t.Fatal("expected nil after removal, got session")
	}
	if srv.SessionCount() != 0 {
		t.Fatalf("expected 0 sessions, got %d", srv.SessionCount())
	}
}

func TestSessionRegistry_SessionCount(t *testing.T) {
	srv := newTestServer()

	if srv.SessionCount() != 0 {
		t.Fatalf("expected 0, got %d", srv.SessionCount())
	}

	sessions := make([]*Session, 5)
	for i := range sessions {
		sessions[i] = newTestSession()
		srv.mu.Lock()
		srv.sessions[sessions[i].ID] = sessions[i]
		srv.mu.Unlock()
	}

	if srv.SessionCount() != 5 {
		t.Fatalf("expected 5, got %d", srv.SessionCount())
	}
}

func TestSessionRegistry_ConcurrentAccess(t *testing.T) {
	srv := newTestServer()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent writes
	sessions := make([]*Session, goroutines)
	for i := 0; i < goroutines; i++ {
		sessions[i] = newTestSession()
	}

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			srv.mu.Lock()
			srv.sessions[sessions[idx].ID] = sessions[idx]
			srv.mu.Unlock()
		}(i)
	}

	// Concurrent reads
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_ = srv.Session(sessions[idx].ID)
			_ = srv.SessionCount()
		}(i)
	}

	wg.Wait()

	if srv.SessionCount() != goroutines {
		t.Fatalf("expected %d sessions, got %d", goroutines, srv.SessionCount())
	}
}

func TestWithConnectionPool(t *testing.T) {
	srv := NewServer(":0", nil, func(s *Session) *dom.Node { return nil }, WithConnectionPool(10))

	if !srv.poolEnabled {
		t.Fatal("expected poolEnabled to be true")
	}
	if srv.poolSize != 10 {
		t.Fatalf("expected poolSize 10, got %d", srv.poolSize)
	}
}

func TestWithSSR(t *testing.T) {
	srv := NewServer(":0", nil, func(s *Session) *dom.Node { return nil }, WithSSR())

	if !srv.ssrEnabled {
		t.Fatal("expected ssrEnabled to be true")
	}
}

func TestWithConnectionPool_Zero(t *testing.T) {
	srv := NewServer(":0", nil, func(s *Session) *dom.Node { return nil }, WithConnectionPool(0))

	if !srv.poolEnabled {
		t.Fatal("expected poolEnabled to be true")
	}
	if srv.poolSize != 0 {
		t.Fatalf("expected poolSize 0, got %d", srv.poolSize)
	}
}
