# Bytewire

A server-driven UI framework written in pure Go. Build interactive web applications with no JavaScript, no JSON APIs, and no hydration.

Bytewire renders UI entirely on the server, serializes DOM mutations as compact binary opcodes, and streams them over WebTransport (HTTP/3) to a thin WASM client that patches the browser DOM directly.

## How It Works

```
Server (Go)                          Browser
┌─────────────┐    binary opcodes    ┌──────────────┐
│ Signals      │ ──────────────────> │ WASM Client  │
│ Virtual DOM  │    WebTransport     │ DOM Patching  │
│ Components   │ <────────────────── │ User Events   │
└─────────────┘    user intents      └──────────────┘
```

- **Server owns all state.** Signals, routing, and business logic live on the server.
- **Binary protocol.** Every DOM mutation is 1-10 bytes. No JSON, no HTML serialization.
- **Zero hydration.** The WASM client only patches the DOM. No re-execution of server logic.
- **Go native.** UI is expressed as Go functions. The compiler is the linter.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "os/signal"

    "github.com/bytewiredev/bytewire/pkg/dom"
    "github.com/bytewiredev/bytewire/pkg/engine"
    "github.com/bytewiredev/bytewire/pkg/style"
)

func app(s *engine.Session) *dom.Node {
    count := dom.NewSignal(0)

    return dom.Div(
        dom.Class(style.Classes(style.Flex, style.FlexCol, style.ItemsCenter, style.P8)),
        dom.Children(
            dom.H1(dom.Children(dom.Text("Counter"))),
            dom.Div(dom.Children(
                dom.TextF(count, func(v int) string {
                    return fmt.Sprintf("Count: %d", v)
                }),
            )),
            dom.Button(
                dom.Class(style.Classes(style.BgBlue500, style.TextWhite, style.Px4, style.Py2, style.RoundedMd)),
                dom.Children(dom.Text("Increment")),
                dom.OnClick(func(_ []byte) {
                    count.Update(func(v int) int { return v + 1 })
                }),
            ),
        ),
    )
}

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    srv := engine.NewServer(":4433", nil, app, engine.WithLogger(logger))

    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
    defer cancel()

    if err := srv.ListenAndServe(ctx); err != nil {
        logger.Error("server error", "error", err)
    }
}
```

## Features

### Reactive Primitives
- **Signals** — `Signal[T]` and `ListSignal[T]` with automatic dirty tracking
- **Reactive DOM** — `TextF`, `ClassF`, `StyleF`, `AttrF` bind signals to DOM attributes
- **Control flow** — `If` and `For` for conditional and list rendering

### Component Library (`pkg/components`)
- **Forms** — TextInput, Checkbox, Select, TextArea with two-way signal binding
- **Table** — Generic `Table[T]` with keyed rows and column definitions
- **Modal** — Overlay with backdrop, controlled by `Signal[bool]`
- **Display** — Card, Badge, Alert, Spinner

### Server Engine
- **WebTransport** (HTTP/3) with **WebSocket fallback** for older browsers
- **Rate limiting** — per-session token bucket throttling
- **Connection pooling** — session registry with pool size enforcement
- **Metrics** — Prometheus-compatible `/metrics` endpoint

### Developer Experience
- **CLI** — `bytewire build` compiles WASM, `bytewire serve` watches and rebuilds
- **Error overlay** — server panics displayed in browser with auto-dismiss
- **DevTools** — `window.__bytewire` console API for state inspection
- **Hot reload** — file watcher with reconnection on server restart

### Styling
- **Type-safe CSS** — Tailwind-style utility classes as Go constants
- **Dead-code elimination** — only referenced classes are emitted
- **Reactive classes** — `ClassF` binds class attributes to signals

### Routing
- **Server-side router** — pattern matching with `:param` segments
- **Browser integration** — popstate, link interception, query string parsing
- **`OpPushHistory`** — server-driven URL updates via History API

## Project Structure

```
cmd/bytewire/       CLI tool (build, serve)
pkg/
  components/       Reusable UI components (form, table, modal, display)
  dom/              Virtual DOM, signals, reactive primitives, diffing
  engine/           WebTransport/WebSocket server, sessions, flush loop
  metrics/          Prometheus-compatible metrics (Counter, Gauge, Registry)
  protocol/         Binary opcode encoding/decoding
  ratelimit/        Token bucket rate limiter
  router/           Server-side routing with param extraction
  style/            Tailwind-style CSS generation
  wasm/             Browser WASM client (DOM patching, transport, devtools)
```

## Binary Protocol

Every DOM mutation is a compact binary frame:

| Opcode | Hex | Description |
|--------|-----|-------------|
| InsertNode | `0x01` | Create element with tag, parent, attributes |
| UpdateText | `0x02` | Set text content of a node |
| RemoveNode | `0x03` | Remove a node from the tree |
| SetAttr | `0x04` | Set an attribute on a node |
| RemoveAttr | `0x05` | Remove an attribute |
| ReplaceText | `0x06` | Surgical text splice |
| SetStyle | `0x07` | Set inline style property |
| PushHistory | `0x08` | Update browser URL |
| Batch | `0x09` | Atomic group of opcodes |
| Error | `0x0A` | Display error overlay |
| DevToolsState | `0x0B` | Session state snapshot |

## Server Options

```go
engine.NewServer(addr, tlsConfig, component,
    engine.WithLogger(logger),
    engine.WithStaticDir("static"),
    engine.WithCSS(style.GenerateAll()),
    engine.WithRateLimit(30, 60),          // 30/s, burst 60
    engine.WithMetrics(metricsRegistry),   // /metrics endpoint
    engine.WithConnectionPool(1000),       // max 1000 sessions
    engine.WithWebSocketFallback(),        // enable WS fallback
)
```

## Requirements

- Go 1.24+
- Browser with WebTransport support (falls back to WebSocket)

## Development

```bash
make build    # Build the CLI
make test     # Run all tests
make wasm     # Compile WASM client
make lint     # Run go vet
make clean    # Remove build artifacts
```

## Examples

See [bytewiredev/examples](https://github.com/bytewiredev/examples) for working applications:
- **counter** — Minimal signal-driven counter
- **pages** — Multi-page app with router
- **todo** — Full CRUD with reactive lists

## License

MIT
