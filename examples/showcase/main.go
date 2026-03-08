// Showcase is a multi-page Bytewire demo app demonstrating components,
// charts, and routing — all in pure Go with no JavaScript.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bytewiredev/bytewire/examples/showcase/pages"
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/router"
	"github.com/bytewiredev/bytewire/pkg/style"
)

func app(s *engine.Session) *dom.Node {
	r := router.New()
	r.Handle("/", func(s *engine.Session) *dom.Node {
		return Layout(s.CurrentPath(), pages.Home(s))
	})
	r.Handle("/components", func(s *engine.Session) *dom.Node {
		return Layout(s.CurrentPath(), pages.Components(s))
	})
	r.Handle("/charts", func(s *engine.Session) *dom.Node {
		return Layout(s.CurrentPath(), pages.Charts(s))
	})
	r.Handle("/about", func(s *engine.Session) *dom.Node {
		return Layout(s.CurrentPath(), pages.About(s))
	})
	r.NotFound(func(s *engine.Session) *dom.Node {
		return Layout(s.CurrentPath(), dom.Div(
			dom.Children(
				dom.H2(
					dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900)),
					dom.Children(dom.Text("404 — Page Not Found")),
				),
			),
		))
	})

	return r.Mount(s)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	css := style.GenerateAll()

	srv := engine.NewServer(":4433", nil, app,
		engine.WithLogger(logger),
		engine.WithCSS(css),
		engine.WithStaticDir("static"),
		engine.WithWebSocketFallback(),
		engine.WithSSR(),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := srv.ListenAndServe(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
