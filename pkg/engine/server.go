package engine

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/bytewiredev/bytewire/pkg/devcert"
	"github.com/bytewiredev/bytewire/pkg/metrics"
	"github.com/bytewiredev/bytewire/pkg/protocol"
	"github.com/bytewiredev/bytewire/pkg/ratelimit"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/net/websocket"
)

// Server is the Bytewire WebTransport server that accepts connections
// and spawns per-user sessions.
type Server struct {
	addr        string
	tlsConfig   *tls.Config
	component   Component
	logger      *slog.Logger
	checkOrigin func(r *http.Request) bool

	// Options for the integrated page server.
	staticDir string
	css       string
	httpAddr  string

	mu       sync.RWMutex
	sessions map[SessionID]*Session
	wt       *webtransport.Server

	// Rate limit config applied to each new session.
	rlRate  float64
	rlBurst int

	// Connection pooling options.
	poolEnabled bool
	poolSize    int

	// WebSocket fallback for browsers without WebTransport support.
	wsFallback bool

	// Metrics registry and default metric handles.
	metricsRegistry *metrics.Registry
	metrics         *metrics.Defaults
}

// ServerOption configures the Server.
type ServerOption func(*Server)

// WithLogger sets a custom logger.
func WithLogger(l *slog.Logger) ServerOption {
	return func(s *Server) { s.logger = l }
}

// WithCheckOrigin sets a custom origin check for WebTransport upgrades.
// By default, webtransport-go enforces same-origin. Use this to allow
// cross-origin connections (e.g. when the static server runs on a different port).
func WithCheckOrigin(fn func(r *http.Request) bool) ServerOption {
	return func(s *Server) { s.checkOrigin = fn }
}

// WithStaticDir serves files from the given directory under /static/.
func WithStaticDir(path string) ServerOption {
	return func(s *Server) { s.staticDir = path }
}

// WithCSS injects CSS into the HTML shell's <style> tag.
func WithCSS(css string) ServerOption {
	return func(s *Server) { s.css = css }
}

// WithHTTPAddr sets the plain HTTP address for the page server (default ":8080").
func WithHTTPAddr(addr string) ServerOption {
	return func(s *Server) { s.httpAddr = addr }
}

// WithRateLimit sets per-session intent rate limiting. Rate is the sustained
// intents per second; burst is the maximum number of intents allowed in a
// quick burst. A zero rate disables rate limiting.
func WithRateLimit(rate float64, burst int) ServerOption {
	return func(s *Server) {
		s.rlRate = rate
		s.rlBurst = burst
	}
}

// WithConnectionPool enables connection pooling with session multiplexing.
// The size parameter sets the maximum number of concurrent sessions allowed.
// A size of 0 or less means unlimited.
func WithConnectionPool(size int) ServerOption {
	return func(s *Server) {
		s.poolEnabled = true
		s.poolSize = size
	}
}

// WithWebSocketFallback enables a /bw-ws WebSocket endpoint as a fallback
// transport for browsers that do not support WebTransport.
func WithWebSocketFallback() ServerOption {
	return func(s *Server) { s.wsFallback = true }
}

// WithMetrics attaches a metrics registry to the server. The default Bytewire
// metrics are registered automatically and the /metrics HTTP endpoint is served
// on the page server mux.
func WithMetrics(r *metrics.Registry) ServerOption {
	return func(s *Server) {
		s.metricsRegistry = r
		s.metrics = metrics.RegisterDefaults(r)
	}
}

// Session returns the session with the given ID, or nil if not found.
func (s *Server) Session(id SessionID) *Session {
	s.mu.RLock()
	sess := s.sessions[id]
	s.mu.RUnlock()
	return sess
}

// SessionCount returns the number of active sessions.
func (s *Server) SessionCount() int {
	s.mu.RLock()
	n := len(s.sessions)
	s.mu.RUnlock()
	return n
}

// NewServer creates a Bytewire server listening on addr.
// If tlsConfig is nil, ListenAndServe will auto-generate an ephemeral dev cert.
func NewServer(addr string, tlsConfig *tls.Config, comp Component, opts ...ServerOption) *Server {
	s := &Server{
		addr:      addr,
		tlsConfig: tlsConfig,
		component: comp,
		logger:    slog.Default(),
		rlRate:    30,
		rlBurst:   60,
		httpAddr:  ":8080",
		sessions:  make(map[SessionID]*Session),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Setup registers the WebTransport /bw handler on the given mux and
// initializes the internal webtransport.Server. Call this when you want
// to control the mux and server lifecycle yourself.
func (s *Server) Setup(mux *http.ServeMux) {
	s.wt = &webtransport.Server{
		H3: &http3.Server{
			Addr:      s.addr,
			TLSConfig: http3.ConfigureTLSConfig(s.tlsConfig),
			Handler:   mux,
		},
		CheckOrigin: s.checkOrigin,
	}
	webtransport.ConfigureHTTP3Server(s.wt.H3)

	mux.HandleFunc("/bw", func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.wt.Upgrade(w, r)
		if err != nil {
			s.logger.Error("webtransport upgrade failed", "error", err)
			return
		}
		s.handleConnection(r.Context(), sess)
	})

	if s.wsFallback {
		mux.Handle("/bw-ws", websocket.Handler(s.handleWebSocket))
	}
}

// ServeUDP starts the WebTransport server on an existing UDP connection.
// Setup must be called first.
func (s *Server) ServeUDP(conn net.PacketConn) error {
	s.logger.Info("Bytewire server starting (UDP)", "addr", s.addr)
	return s.wt.Serve(conn)
}

// ListenAndServe starts both the WebTransport (UDP) and plain HTTP (TCP) servers.
// If no TLS config was provided, an ephemeral dev cert is auto-generated.
// The plain HTTP server serves the HTML shell, static files, and WASM bootstrap.
// Blocks until ctx is cancelled, then shuts down both servers.
func (s *Server) ListenAndServe(ctx context.Context) error {
	// Auto-generate cert if none provided.
	var certHashJS string
	if s.tlsConfig == nil {
		cert, hash, err := devcert.Generate()
		if err != nil {
			return fmt.Errorf("devcert: %w", err)
		}
		s.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		certHashJS = devcert.FormatHashJS(hash)
		s.logger.Info("dev cert generated", "sha256", fmt.Sprintf("%x", hash))
	}

	// Cross-origin is needed because the page server and WebTransport are on different ports.
	if s.checkOrigin == nil {
		s.checkOrigin = func(r *http.Request) bool { return true }
	}

	// Setup WebTransport on a shared mux.
	wtMux := http.NewServeMux()
	s.Setup(wtMux)

	// Start WebTransport UDP server.
	udpAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return fmt.Errorf("resolve UDP: %w", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}
	go func() {
		if err := s.ServeUDP(udpConn); err != nil {
			s.logger.Error("WebTransport server error", "error", err)
		}
	}()

	// Build and serve HTML shell on plain HTTP.
	shell := buildShellHTML(certHashJS, s.css, s.addr)
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, shell)
	})
	if s.staticDir != "" {
		httpMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticDir))))
	}
	if s.wsFallback {
		httpMux.Handle("/bw-ws", websocket.Handler(s.handleWebSocket))
	}
	if s.metricsRegistry != nil {
		httpMux.Handle("/metrics", s.metricsRegistry.Handler())
	}

	httpSrv := &http.Server{Addr: s.httpAddr, Handler: httpMux}
	go func() {
		s.logger.Info("HTTP page server starting", "addr", "http://localhost"+s.httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	s.logger.Info("Bytewire server starting", "wtAddr", s.addr, "httpAddr", s.httpAddr)
	<-ctx.Done()
	s.logger.Info("shutting down...")

	httpSrv.Close()
	return s.Shutdown()
}

// newLimiter creates a rate limiter using the server's config.
// Returns nil if rate limiting is disabled (rate <= 0).
func (s *Server) newLimiter() *ratelimit.Limiter {
	if s.rlRate <= 0 {
		return nil
	}
	return ratelimit.New(s.rlRate, s.rlBurst)
}

// handleConnection manages a single WebTransport session.
func (s *Server) handleConnection(ctx context.Context, wtSession *webtransport.Session) {
	// Enforce pool size limit if connection pooling is enabled.
	if s.poolEnabled && s.poolSize > 0 {
		s.mu.RLock()
		count := len(s.sessions)
		s.mu.RUnlock()
		if count >= s.poolSize {
			s.logger.Warn("connection pool full, rejecting session", "poolSize", s.poolSize)
			wtSession.CloseWithError(429, "connection pool full")
			return
		}
	}

	w := &wtWriter{session: wtSession}
	sess := NewSession(ctx, w, s.logger, s.newLimiter(), s.metrics)

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	if s.metrics != nil {
		s.metrics.SessionsTotal.Inc()
		s.metrics.SessionsActive.Inc()
	}
	s.logger.Info("session connected", "sessionID", sess.ID)

	// Send protocol version handshake.
	helloBuf := protocol.AcquireBuffer()
	helloBuf.EncodeHello(protocol.ProtocolMajor, protocol.ProtocolMinor)
	if err := w.WriteMessage(helloBuf.Bytes()); err != nil {
		helloBuf.Release()
		s.logger.Error("failed to send hello", "error", err)
		return
	}
	helloBuf.Release()

	if err := sess.Mount(s.component); err != nil {
		s.logger.Error("mount failed", "error", err)
		sess.Close()
		if s.metrics != nil {
			s.metrics.SessionsActive.Dec()
		}
		return
	}

	// Read client intents from bidirectional stream
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.sessions, sess.ID)
			s.mu.Unlock()
			sess.Close()
			if s.metrics != nil {
				s.metrics.SessionsActive.Dec()
			}
			s.logger.Info("session disconnected", "sessionID", sess.ID)
		}()

		for {
			stream, err := wtSession.AcceptStream(sess.Context())
			if err != nil {
				return
			}
			go s.handleStream(sess, stream)
		}
	}()
}

// handleStream reads length-prefixed binary frames from a client stream.
// Each frame is [4B length][payload]. Multiple frames may arrive on the same stream.
func (s *Server) handleStream(sess *Session, stream *webtransport.Stream) {
	defer stream.Close()
	lenBuf := make([]byte, 4)
	for {
		// Read 4-byte length prefix
		if _, err := io.ReadFull(stream, lenBuf); err != nil {
			return
		}
		frameLen := int(binary.BigEndian.Uint32(lenBuf))
		if frameLen <= 0 || frameLen > 65536 {
			return
		}

		// Read exact frame body
		frame := make([]byte, 4+frameLen)
		copy(frame, lenBuf)
		if _, err := io.ReadFull(stream, frame[4:]); err != nil {
			return
		}

		if err := sess.HandleIntent(frame); err != nil {
			s.logger.Error("intent error", "error", err, "sessionID", sess.ID)
		}
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	for _, sess := range s.sessions {
		sess.Close()
	}
	s.mu.Unlock()
	return s.wt.Close()
}

// wtWriter adapts a WebTransport session to the Writer interface.
type wtWriter struct {
	session *webtransport.Session
}

func (w *wtWriter) WriteMessage(data []byte) error {
	stream, err := w.session.OpenUniStream()
	if err != nil {
		return err
	}
	_, err = stream.Write(data)
	stream.Close()
	return err
}

func (w *wtWriter) Close() error {
	w.session.CloseWithError(0, "session closed")
	return nil
}

// handleWebSocket handles a single WebSocket connection as a fallback transport.
func (s *Server) handleWebSocket(conn *websocket.Conn) {
	conn.PayloadType = websocket.BinaryFrame

	// Enforce pool size limit if connection pooling is enabled.
	if s.poolEnabled && s.poolSize > 0 {
		s.mu.RLock()
		count := len(s.sessions)
		s.mu.RUnlock()
		if count >= s.poolSize {
			s.logger.Warn("connection pool full, rejecting WebSocket session", "poolSize", s.poolSize)
			conn.Close()
			return
		}
	}

	w := &wsWriter{conn: conn}
	sess := NewSession(context.Background(), w, s.logger, s.newLimiter(), s.metrics)

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	if s.metrics != nil {
		s.metrics.SessionsTotal.Inc()
		s.metrics.SessionsActive.Inc()
	}
	s.logger.Info("WebSocket session connected", "sessionID", sess.ID)

	if err := sess.Mount(s.component); err != nil {
		s.logger.Error("mount failed", "error", err)
		sess.Close()
		if s.metrics != nil {
			s.metrics.SessionsActive.Dec()
		}
		return
	}

	// Read length-prefixed frames from WebSocket messages.
	defer func() {
		s.mu.Lock()
		delete(s.sessions, sess.ID)
		s.mu.Unlock()
		sess.Close()
		if s.metrics != nil {
			s.metrics.SessionsActive.Dec()
		}
		s.logger.Info("WebSocket session disconnected", "sessionID", sess.ID)
	}()

	s.readWSFrames(sess, conn)
}

// readWSFrames reads length-prefixed binary frames from a WebSocket connection.
func (s *Server) readWSFrames(sess *Session, conn *websocket.Conn) {
	lenBuf := make([]byte, 4)
	for {
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		frameLen := int(binary.BigEndian.Uint32(lenBuf))
		if frameLen <= 0 || frameLen > 65536 {
			return
		}

		frame := make([]byte, 4+frameLen)
		copy(frame, lenBuf)
		if _, err := io.ReadFull(conn, frame[4:]); err != nil {
			return
		}

		if err := sess.HandleIntent(frame); err != nil {
			s.logger.Error("intent error", "error", err, "sessionID", sess.ID)
		}
	}
}

// wsWriter adapts a WebSocket connection to the Writer interface.
type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) WriteMessage(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.conn.Write(data)
	return err
}

func (w *wsWriter) Close() error {
	return w.conn.Close()
}

// buildShellHTML returns the HTML page that bootstraps the WASM client.
// It injects the cert hash and CSS so the WASM binary is portable.
func buildShellHTML(certHashJS, css, wtAddr string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Bytewire App</title>
  <style>
button{cursor:pointer;user-select:none;-webkit-user-select:none;border:none;outline:none}
button:active{opacity:0.8;transform:scale(0.97)}
%s</style>
</head>
<body>
  <div id="bw-root"></div>
  <script>
    window.__bw_config = {
      url: "https://localhost%s/bw",
      certHash: new Uint8Array(%s),
      wsURL: "ws://" + location.host + "/bw-ws"
    };
  </script>
  <script src="/static/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/static/bytewire.wasm"), go.importObject)
      .then(result => go.run(result.instance))
      .catch(err => {
        document.getElementById("bw-root").textContent = "WASM load failed: " + err.message;
      });
  </script>
</body>
</html>`, css, wtAddr, certHashJS)
}
