package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCounterIncAndValue(t *testing.T) {
	c := &Counter{Name: "test_counter", Help: "A test counter"}
	if c.Value() != 0 {
		t.Fatalf("expected 0, got %d", c.Value())
	}
	c.Inc()
	c.Inc()
	c.Inc()
	if c.Value() != 3 {
		t.Fatalf("expected 3, got %d", c.Value())
	}
}

func TestGaugeIncDecAndValue(t *testing.T) {
	g := &Gauge{Name: "test_gauge", Help: "A test gauge"}
	g.Inc()
	g.Inc()
	g.Dec()
	if g.Value() != 1 {
		t.Fatalf("expected 1, got %d", g.Value())
	}
	g.Dec()
	g.Dec()
	if g.Value() != -1 {
		t.Fatalf("expected -1, got %d", g.Value())
	}
}

func TestRegistryDeduplicates(t *testing.T) {
	r := NewRegistry()
	c1 := r.Counter("dup", "first")
	c2 := r.Counter("dup", "second")
	if c1 != c2 {
		t.Fatal("expected same counter for duplicate name")
	}
	g1 := r.Gauge("gdup", "first")
	g2 := r.Gauge("gdup", "second")
	if g1 != g2 {
		t.Fatal("expected same gauge for duplicate name")
	}
}

func TestHandlerPrometheusFormat(t *testing.T) {
	r := NewRegistry()
	c := r.Counter("http_requests_total", "Total HTTP requests")
	g := r.Gauge("goroutines_active", "Active goroutines")
	c.Inc()
	c.Inc()
	g.Inc()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	r.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("unexpected content-type: %s", ct)
	}

	expected := []string{
		"# HELP http_requests_total Total HTTP requests",
		"# TYPE http_requests_total counter",
		"http_requests_total 2",
		"# HELP goroutines_active Active goroutines",
		"# TYPE goroutines_active gauge",
		"goroutines_active 1",
	}
	for _, line := range expected {
		if !strings.Contains(body, line) {
			t.Errorf("missing line: %q\nbody:\n%s", line, body)
		}
	}
}

func TestRegisterDefaults(t *testing.T) {
	r := NewRegistry()
	d := RegisterDefaults(r)
	d.SessionsTotal.Inc()
	d.SessionsActive.Inc()
	d.IntentsTotal.Inc()
	d.IntentsDropped.Inc()
	d.FlushTotal.Inc()
	d.ErrorsTotal.Inc()

	if d.SessionsTotal.Value() != 1 {
		t.Fatal("sessions_total not incremented")
	}
	if d.SessionsActive.Value() != 1 {
		t.Fatal("sessions_active not incremented")
	}
}
