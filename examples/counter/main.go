// Counter is a minimal CBS example: a button that increments a number.
// The entire UI is pure Go — no HTML templates, no JavaScript, no JSON API.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/protocol"
	"github.com/bytewiredev/bytewire/pkg/style"
)

func counterApp(s *engine.Session) *dom.Node {
	count := dom.NewSignal(0)

	return dom.Div(
		dom.Class(style.Classes(style.Flex, style.FlexCol, style.ItemsCenter, style.Gap4, style.P8)),
		dom.Children(
			dom.H1(
				dom.Class(style.Classes(style.Text3Xl, style.FontBold, style.TextGray900)),
				dom.Children(dom.Text("CBS Counter")),
			),
			dom.Div(
				dom.Class(style.Classes(style.Text2Xl, style.FontMedium)),
				dom.Children(
					dom.TextF(count, func(v int) string {
						return fmt.Sprintf("Count: %d", v)
					}),
				),
			),
			dom.Div(
				dom.Class(style.Classes(style.Flex, style.Gap4)),
				dom.Children(
					dom.Button(
						dom.Class(style.Classes(
							style.BgBlue500, style.TextWhite, style.Px4, style.Py2,
							style.RoundedMd, style.FontMedium, style.ShadowMd,
						)),
						dom.Children(dom.Text("Increment")),
						dom.OnClick(func(_ []byte) {
							count.Update(func(v int) int { return v + 1 })
						}),
					),
					dom.Button(
						dom.Class(style.Classes(
							style.BgRed500, style.TextWhite, style.Px4, style.Py2,
							style.RoundedMd, style.FontMedium, style.ShadowMd,
						)),
						dom.Children(dom.Text("Reset")),
						dom.OnClick(func(_ []byte) {
							count.Set(0)
						}),
					),
				),
			),
		),
	)
}

func main() {
	// For development, use a self-signed TLS cert.
	// In production, use Let's Encrypt or your own CA.
	tlsConfig := &tls.Config{
		// TLS 1.3 only — as per the CBS manifesto
		MinVersion: tls.VersionTLS13,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	srv := engine.NewServer(":4433", tlsConfig, counterApp, engine.WithLogger(logger))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down...")
		srv.Shutdown()
	}()

	if err := srv.ListenAndServe(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// Ensure protocol import is used (referenced by event constants via dom.OnClick)
var _ = protocol.EventClick
