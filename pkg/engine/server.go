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

	"github.com/cbsframework/cbs/pkg/devcert"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// Server is the CBS WebTransport server that accepts connections
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

	mu       sync.Mutex
	sessions map[SessionID]*Session
	wt       *webtransport.Server
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

// NewServer creates a CBS server listening on addr.
// If tlsConfig is nil, ListenAndServe will auto-generate an ephemeral dev cert.
func NewServer(addr string, tlsConfig *tls.Config, comp Component, opts ...ServerOption) *Server {
	s := &Server{
		addr:      addr,
		tlsConfig: tlsConfig,
		component: comp,
		logger:    slog.Default(),
		httpAddr:  ":8080",
		sessions:  make(map[SessionID]*Session),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Setup registers the WebTransport /cbs handler on the given mux and
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

	mux.HandleFunc("/cbs", func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.wt.Upgrade(w, r)
		if err != nil {
			s.logger.Error("webtransport upgrade failed", "error", err)
			return
		}
		s.handleConnection(r.Context(), sess)
	})
}

// ServeUDP starts the WebTransport server on an existing UDP connection.
// Setup must be called first.
func (s *Server) ServeUDP(conn net.PacketConn) error {
	s.logger.Info("CBS server starting (UDP)", "addr", s.addr)
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

	httpSrv := &http.Server{Addr: s.httpAddr, Handler: httpMux}
	go func() {
		s.logger.Info("HTTP page server starting", "addr", "http://localhost"+s.httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server error", "error", err)
		}
	}()

	s.logger.Info("CBS server starting", "wtAddr", s.addr, "httpAddr", s.httpAddr)
	<-ctx.Done()
	s.logger.Info("shutting down...")

	httpSrv.Close()
	return s.Shutdown()
}

// handleConnection manages a single WebTransport session.
func (s *Server) handleConnection(ctx context.Context, wtSession *webtransport.Session) {
	w := &wtWriter{session: wtSession}
	sess := NewSession(ctx, w, s.logger)

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	s.logger.Info("session connected", "sessionID", sess.ID)

	if err := sess.Mount(s.component); err != nil {
		s.logger.Error("mount failed", "error", err)
		sess.Close()
		return
	}

	// Read client intents from bidirectional stream
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.sessions, sess.ID)
			s.mu.Unlock()
			sess.Close()
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

// buildShellHTML returns the HTML page that bootstraps the WASM client.
// It injects the cert hash and CSS so the WASM binary is portable.
func buildShellHTML(certHashJS, css, wtAddr string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CBS App</title>
  <style>
button{cursor:pointer;user-select:none;-webkit-user-select:none;border:none;outline:none}
button:active{opacity:0.8;transform:scale(0.97)}
%s</style>
</head>
<body>
  <div id="cbs-root"></div>
  <script>
    window.__cbs_config = {
      url: "https://localhost%s/cbs",
      certHash: new Uint8Array(%s)
    };
  </script>
  <script src="/static/wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("/static/cbs.wasm"), go.importObject)
      .then(result => go.run(result.instance))
      .catch(err => {
        document.getElementById("cbs-root").textContent = "WASM load failed: " + err.message;
      });
  </script>
</body>
</html>`, css, wtAddr, certHashJS)
}
