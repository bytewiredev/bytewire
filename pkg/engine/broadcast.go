package engine

import "sync"

// Broadcast enables pub/sub messaging across sessions.
// Use this for features like live notifications or collaborative editing.
type Broadcast struct {
	mu          sync.RWMutex
	subscribers map[string][]chan []byte
}

// NewBroadcast creates a new broadcast hub.
func NewBroadcast() *Broadcast {
	return &Broadcast{
		subscribers: make(map[string][]chan []byte),
	}
}

// Subscribe returns a channel that receives messages for the given topic.
func (b *Broadcast) Subscribe(topic string) chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	b.mu.Unlock()
	return ch
}

// Publish sends data to all subscribers of a topic.
func (b *Broadcast) Publish(topic string, data []byte) {
	b.mu.RLock()
	subs := b.subscribers[topic]
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- data:
		default:
			// Drop if subscriber is slow — prevents backpressure from blocking.
		}
	}
}

// Unsubscribe removes a channel from a topic.
func (b *Broadcast) Unsubscribe(topic string, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[topic]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}
