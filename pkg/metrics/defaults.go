package metrics

// Defaults holds the standard set of Bytewire server metrics.
type Defaults struct {
	SessionsTotal   *Counter
	SessionsActive  *Gauge
	IntentsTotal    *Counter
	IntentsDropped  *Counter
	FlushTotal      *Counter
	ErrorsTotal     *Counter
}

// RegisterDefaults registers the standard Bytewire metrics on the given
// registry and returns convenience handles.
func RegisterDefaults(r *Registry) *Defaults {
	return &Defaults{
		SessionsTotal:   r.Counter("bytewire_sessions_total", "Total sessions created"),
		SessionsActive:  r.Gauge("bytewire_sessions_active", "Currently active sessions"),
		IntentsTotal:    r.Counter("bytewire_intents_total", "Total intents processed"),
		IntentsDropped:  r.Counter("bytewire_intents_dropped", "Intents dropped by rate limiter"),
		FlushTotal:      r.Counter("bytewire_flush_total", "Total DOM flushes"),
		ErrorsTotal:     r.Counter("bytewire_errors_total", "Total error overlays sent"),
	}
}
