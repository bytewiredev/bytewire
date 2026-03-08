package pages

import (
	"github.com/bytewiredev/bytewire/pkg/dom"
	"github.com/bytewiredev/bytewire/pkg/engine"
	"github.com/bytewiredev/bytewire/pkg/style"
)

// About renders static content about the Bytewire framework.
func About(_ *engine.Session) *dom.Node {
	features := []string{
		"Binary protocol over WebTransport — zero JSON, zero HTML parsing",
		"Server-driven reactive UI — all state lives on the server in Go",
		"Type-safe signals — Signal[T], ListSignal[T], BoxSignal[T], Computed",
		"LIS-based list reconciliation — minimum DOM moves",
		"OpSwapNodes — dedicated 9-byte opcode for 2-element swaps",
		"OpBatchText — batched text updates in a single frame",
		"Hover prefetch — pre-render routes on mouseover for instant navigation",
		"Hybrid canvas charts — server-defined, client-rendered via Canvas 2D",
		"Server-side rendering (SSR) with hydration",
		"WebSocket fallback for browsers without WebTransport",
		"Offline queue with IndexedDB persistence",
		"WebAuthn passkey authentication",
	}

	featureNodes := make([]*dom.Node, 0, len(features))
	for _, f := range features {
		featureNodes = append(featureNodes, dom.Li(
			dom.Class(style.Classes(style.Py1)),
			dom.Children(dom.Text(f)),
		))
	}

	return dom.Div(
		dom.Children(
			dom.H2(
				dom.Class(style.Classes(style.Text2Xl, style.FontBold, style.TextGray900, style.Mb6)),
				dom.Children(dom.Text("About Bytewire")),
			),

			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P6, style.Mb6)),
				dom.Children(
					dom.H3(
						dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextGray900, style.Mb4)),
						dom.Children(dom.Text("What is Bytewire?")),
					),
					dom.P(
						dom.Class(style.Classes(style.TextGray700, style.Mb4)),
						dom.Children(dom.Text(
							"Bytewire is a Go framework for building server-driven reactive UIs. "+
								"Instead of shipping JavaScript to the browser, Bytewire keeps all application "+
								"state on the server and streams minimal binary DOM patches over WebTransport. "+
								"The browser runs a tiny WASM client that applies patches and delegates events.")),
					),
					dom.P(
						dom.Class(style.Classes(style.TextGray700)),
						dom.Children(dom.Text(
							"The result: Go developers can build full-stack web applications in pure Go — "+
								"no HTML templates, no JSON APIs, no JavaScript build tools. Just Go.")),
					),
				),
			),

			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P6, style.Mb6)),
				dom.Children(
					dom.H3(
						dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextGray900, style.Mb4)),
						dom.Children(dom.Text("Key Features")),
					),
					dom.Ul(
						dom.Class(style.Classes(style.TextGray700, style.TextSm)),
						dom.Children(featureNodes...),
					),
				),
			),

			dom.Div(
				dom.Class(style.Classes(style.BgWhite, style.RoundedLg, style.ShadowMd, style.P6)),
				dom.Children(
					dom.H3(
						dom.Class(style.Classes(style.TextXl, style.FontBold, style.TextGray900, style.Mb4)),
						dom.Children(dom.Text("Architecture")),
					),
					dom.P(
						dom.Class(style.Classes(style.TextGray700, style.Mb4)),
						dom.Children(dom.Text(
							"Server: Go application creates a virtual DOM tree with reactive signals. "+
								"When signals change, the engine computes minimal binary diffs and streams "+
								"them over WebTransport (QUIC/HTTP3) or WebSocket fallback.")),
					),
					dom.P(
						dom.Class(style.Classes(style.TextGray700)),
						dom.Children(dom.Text(
							"Client: A ~15KB WASM binary receives binary opcodes and applies DOM "+
								"mutations directly. Event delegation captures user interactions and sends "+
								"compact binary intents back to the server. Charts use a hybrid approach: "+
								"the server defines data via attributes, the client renders via Canvas 2D.")),
					),
				),
			),
		),
	)
}
