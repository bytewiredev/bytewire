package engine

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// Server is the CBS WebTransport server that accepts connections
// and spawns per-user sessions.
type Server struct {
	addr      string
	tlsConfig *tls.Config
	component Component
	logger    *slog.Logger

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

// NewServer creates a CBS server listening on addr.
// The tlsConfig must include a valid certificate for WebTransport (HTTP/3).
func NewServer(addr string, tlsConfig *tls.Config, comp Component, opts ...ServerOption) *Server {
	s := &Server{
		addr:      addr,
		tlsConfig: tlsConfig,
		component: comp,
		logger:    slog.Default(),
		sessions:  make(map[SessionID]*Session),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ListenAndServe starts the HTTP/3 server with WebTransport upgrade support.
func (s *Server) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()

	s.wt = &webtransport.Server{
		H3: http3.Server{
			Addr:      s.addr,
			TLSConfig: s.tlsConfig,
			Handler:   mux,
		},
	}

	mux.HandleFunc("/cbs", func(w http.ResponseWriter, r *http.Request) {
		sess, err := s.wt.Upgrade(w, r)
		if err != nil {
			s.logger.Error("webtransport upgrade failed", "error", err)
			return
		}
		s.handleConnection(ctx, sess)
	})

	// Serve a minimal HTML page that loads the WASM client
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, shellHTML)
	})

	s.logger.Info("CBS server starting", "addr", s.addr)
	return s.wt.ListenAndServe()
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

// handleStream reads binary messages from a client stream.
func (s *Server) handleStream(sess *Session, stream webtransport.Stream) {
	defer stream.Close()
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err != nil {
			return
		}
		if err := sess.HandleIntent(buf[:n]); err != nil {
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

// shellHTML is the minimal HTML page served to bootstrap the WASM client.
const shellHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CBS App</title>
</head>
<body>
  <div id="cbs-root"></div>
  <script src="/cbs.js"></script>
</body>
</html>`
