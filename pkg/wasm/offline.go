//go:build js && wasm

package wasm

import "sync"

// OfflineQueue buffers user intents when the WebTransport connection drops.
// Queued messages are flushed when the connection is restored.
type OfflineQueue struct {
	mu      sync.Mutex
	queue   [][]byte
	maxSize int
}

// NewOfflineQueue creates a queue with a maximum buffer size.
func NewOfflineQueue(maxSize int) *OfflineQueue {
	return &OfflineQueue{
		queue:   make([][]byte, 0, 64),
		maxSize: maxSize,
	}
}

// Enqueue adds a binary message to the offline buffer.
// Returns false if the queue is full.
func (q *OfflineQueue) Enqueue(data []byte) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queue) >= q.maxSize {
		return false
	}

	cp := make([]byte, len(data))
	copy(cp, data)
	q.queue = append(q.queue, cp)
	return true
}

// Flush returns all queued messages and clears the buffer.
func (q *OfflineQueue) Flush() [][]byte {
	q.mu.Lock()
	defer q.mu.Unlock()

	msgs := q.queue
	q.queue = make([][]byte, 0, 64)
	return msgs
}

// Len returns the number of queued messages.
func (q *OfflineQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.queue)
}

// GlobalQueue is the package-level offline queue used by the WASM client.
var GlobalQueue = NewOfflineQueue(256)

// IsOnline tracks whether the client has an active connection.
// Use sync/atomic for goroutine safety.
var IsOnline int32
