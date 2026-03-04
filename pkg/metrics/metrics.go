// Package metrics provides lightweight Prometheus-compatible counters and
// gauges using only the standard library.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
)

// Counter is a monotonically increasing int64 metric.
type Counter struct {
	Name string
	Help string
	val  atomic.Int64
}

// Inc increments the counter by 1.
func (c *Counter) Inc() { c.val.Add(1) }

// Value returns the current counter value.
func (c *Counter) Value() int64 { return c.val.Load() }

// Gauge is an int64 metric that can go up and down.
type Gauge struct {
	Name string
	Help string
	val  atomic.Int64
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() { g.val.Add(1) }

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() { g.val.Add(-1) }

// Value returns the current gauge value.
func (g *Gauge) Value() int64 { return g.val.Load() }

// Registry holds named counters and gauges.
type Registry struct {
	mu       sync.RWMutex
	counters []*Counter
	gauges   []*Gauge
}

// NewRegistry creates an empty metrics registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Counter registers and returns a counter with the given name and help text.
// If a counter with the same name already exists it is returned as-is.
func (r *Registry) Counter(name, help string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.counters {
		if c.Name == name {
			return c
		}
	}
	c := &Counter{Name: name, Help: help}
	r.counters = append(r.counters, c)
	return c
}

// Gauge registers and returns a gauge with the given name and help text.
// If a gauge with the same name already exists it is returned as-is.
func (r *Registry) Gauge(name, help string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, g := range r.gauges {
		if g.Name == name {
			return g
		}
	}
	g := &Gauge{Name: name, Help: help}
	r.gauges = append(r.gauges, g)
	return g
}

// Handler returns an http.Handler that serves all registered metrics in
// Prometheus text exposition format.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		r.mu.RLock()
		// Snapshot slices under lock.
		counters := make([]*Counter, len(r.counters))
		copy(counters, r.counters)
		gauges := make([]*Gauge, len(r.gauges))
		copy(gauges, r.gauges)
		r.mu.RUnlock()

		sort.Slice(counters, func(i, j int) bool { return counters[i].Name < counters[j].Name })
		sort.Slice(gauges, func(i, j int) bool { return gauges[i].Name < gauges[j].Name })

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		for _, c := range counters {
			fmt.Fprintf(w, "# HELP %s %s\n", c.Name, c.Help)
			fmt.Fprintf(w, "# TYPE %s counter\n", c.Name)
			fmt.Fprintf(w, "%s %d\n", c.Name, c.Value())
		}
		for _, g := range gauges {
			fmt.Fprintf(w, "# HELP %s %s\n", g.Name, g.Help)
			fmt.Fprintf(w, "# TYPE %s gauge\n", g.Name)
			fmt.Fprintf(w, "%s %d\n", g.Name, g.Value())
		}
	})
}
