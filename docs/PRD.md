# CBS — Continuous Binary Synchronization

## Product Requirements Document

**Version:** 0.1.0
**Module:** `github.com/bytewiredev/bytewire`
**Go:** 1.24+
**Status:** Alpha
**Last Updated:** 2026-03-04

---

## Table of Contents

1. [Executive Summary & Vision](#1-executive-summary--vision)
2. [Architecture Overview](#2-architecture-overview)
3. [Binary Protocol Specification](#3-binary-protocol-specification)
4. [Reactive State System](#4-reactive-state-system)
5. [Virtual DOM & Component Model](#5-virtual-dom--component-model)
6. [Server Engine](#6-server-engine)
7. [WASM Client Runtime](#7-wasm-client-runtime)
8. [Security Architecture](#8-security-architecture)
9. [Styling System](#9-styling-system)
10. [CLI Toolchain](#10-cli-toolchain)
11. [Offline & Resilience](#11-offline--resilience)
12. [Developer API Reference](#12-developer-api-reference)
13. [Performance Characteristics](#13-performance-characteristics)
14. [Roadmap & Future Work](#14-roadmap--future-work)
15. [Glossary](#15-glossary)

---

## 1. Executive Summary & Vision

### What is CBS?

CBS (Continuous Binary Synchronization) is a server-driven UI framework written in pure Go. It eliminates the traditional client-server split by rendering UI entirely on the server, serializing DOM mutations as compact binary opcodes, and streaming them over WebTransport (HTTP/3 + QUIC) to a thin WASM client that applies them directly to the browser DOM.

### The Problem

Modern web development suffers from an accidental complexity explosion:

- **Duplicated logic** — Validation, routing, and state management are written twice: once on the server, once on the client.
- **JSON tax** — Every interaction round-trips through JSON serialization, HTTP request construction, response parsing, and client-side state reconciliation.
- **JavaScript monoculture** — Teams are forced into JavaScript/TypeScript regardless of backend language preference, doubling the language surface.
- **Hydration theater** — SSR frameworks ship full HTML, then re-execute the same logic client-side to "hydrate" interactivity, wasting bandwidth and CPU.

### The CBS Answer

CBS takes a different approach:

| Traditional SPA | CBS |
|---|---|
| JSON API payloads | Binary opcodes (1-10 bytes per mutation) |
| Client-side rendering | Server-side rendering with binary delta streaming |
| JavaScript framework | Go-only — no JS authoring required |
| HTTP/1.1 or HTTP/2 request-response | WebTransport bidirectional streams (HTTP/3) |
| Template languages (JSX, HTML) | Pure Go function calls |
| npm, webpack, bundlers | `go build` — single binary |

### Design Principles

1. **Server Authority** — The server owns all state. The client is a thin projection surface.
2. **Binary First** — Every byte on the wire is intentional. No JSON, no XML, no HTML serialization.
3. **Go Native** — UI is expressed as Go functions. The compiler is the linter, the type checker, and the dead-code eliminator.
4. **Zero Hydration** — There is no client-side re-execution. The WASM client receives opcodes and patches the DOM. Period.
5. **Secure by Default** — TLS 1.3 mandatory, WebAuthn-native authentication, no `innerHTML` surface for XSS.

### Target Use Cases

- Internal enterprise dashboards with real-time data
- Collaborative applications (shared editing, live notifications)
- Admin panels and CRUD applications
- Any application where server authority over state is desirable

---

## 2. Architecture Overview

### System Topology

```
┌──────────────────────────────────────────────────────────┐
│                      CBS Server (Go)                     │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐              │
│  │ Component │  │ Signal   │  │ Virtual   │              │
│  │ Functions │──│ Graph    │──│ DOM Tree  │              │
│  └──────────┘  └──────────┘  └─────┬─────┘              │
│                                    │ Diff                │
│                              ┌─────▼─────┐              │
│                              │ Binary    │              │
│                              │ Encoder   │              │
│                              └─────┬─────┘              │
│  ┌───────────┐                     │                     │
│  │ Broadcast │                     │                     │
│  │ Pub/Sub   │                     │                     │
│  └───────────┘                     │                     │
│                              ┌─────▼─────┐              │
│                              │  Session  │              │
│                              │  Manager  │              │
│                              └─────┬─────┘              │
└────────────────────────────────────┼─────────────────────┘
                                     │ WebTransport (HTTP/3)
                                     │ TLS 1.3 / QUIC
                                     │
┌────────────────────────────────────┼─────────────────────┐
│                      Browser       │                     │
│                              ┌─────▼─────┐              │
│                              │  WASM     │              │
│                              │  Client   │              │
│                              └─────┬─────┘              │
│                                    │ syscall/js          │
│                              ┌─────▼─────┐              │
│                              │ Browser   │              │
│                              │ DOM       │              │
│                              └───────────┘              │
│  ┌───────────┐  ┌──────────────┐                        │
│  │ Offline   │  │ WebAuthn     │                        │
│  │ Queue     │  │ Passkeys     │                        │
│  └───────────┘  └──────────────┘                        │
└──────────────────────────────────────────────────────────┘
```

### Data Flow: User Click

```
 User clicks button
       │
       ▼
 ┌─────────────┐
 │ WASM Client │  Encode: [0x10][NodeID][0x01 Click][payload]
 └──────┬──────┘
        │ WebTransport stream
        ▼
 ┌─────────────┐
 │ Session     │  Decode binary → dispatch to Node.Handlers[EventClick]
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │ Handler fn  │  signal.Update(func(v int) int { return v + 1 })
 └──────┬──────┘
        │ Signal fires observers
        ▼
 ┌─────────────┐
 │ TextF node  │  Text mutated in-place by observer
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │ Diff Engine │  Compare old vs new tree → emit binary opcodes
 └──────┬──────┘
        │
        ▼
 ┌─────────────┐
 │ Buffer      │  [0x01][NodeID]["Count: 2"]  (10 bytes)
 └──────┬──────┘
        │ WebTransport uni-stream
        ▼
 ┌─────────────┐
 │ WASM Client │  node.Set("textContent", "Count: 2")
 └─────────────┘
```

### Package Dependency Graph

```
cmd/cbs/main.go
    └── (standalone CLI)

examples/counter/main.go
    ├── pkg/dom
    ├── pkg/engine
    ├── pkg/protocol
    └── pkg/style

pkg/engine
    ├── pkg/dom
    ├── pkg/protocol
    ├── quic-go/webtransport-go
    └── quic-go/quic-go

pkg/dom
    └── pkg/protocol

pkg/wasm
    └── pkg/protocol  (via opcode constants)

pkg/style
    └── (no internal dependencies)

pkg/protocol
    └── encoding/binary (stdlib)
```

### Module Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| `github.com/quic-go/quic-go` | v0.59.0 | QUIC transport layer (HTTP/3) |
| `github.com/quic-go/webtransport-go` | v0.10.0 | WebTransport protocol support |
| Go standard library | 1.24+ | Everything else — no other third-party deps |

---

## 3. Binary Protocol Specification

The CBS binary protocol is a compact instruction set for DOM mutations. Every message is a byte sequence designed for zero-copy parsing and minimal allocation.

### Design Goals

- **Minimal wire size** — A text update is 5 + len(text) bytes. No field names, no delimiters, no framing overhead.
- **Zero-copy decoding** — The decoder reads directly from the byte slice without intermediate allocations.
- **Pooled encoding** — Buffers are recycled via `sync.Pool` to eliminate GC pressure on hot paths.
- **Deterministic parsing** — Every opcode has a fixed prefix structure. The decoder never backtracks.

### ISA Reference

#### Server → Client Opcodes (0x01 – 0x0F)

##### `0x01` OpUpdateText

Set the `textContent` of a DOM node.

```
┌──────┬───────────┬─────────────────┐
│ 0x01 │ 4B NodeID │ UTF-8 text      │
│ 1B   │ uint32 BE │ variable length │
└──────┴───────────┴─────────────────┘
```

- **NodeID**: Big-endian `uint32` identifying the target node.
- **Text**: Raw UTF-8 bytes consuming the remainder of the frame.
- **Total size**: 5 + len(text) bytes.

##### `0x02` OpSetAttr

Set an HTML attribute on a DOM node.

```
┌──────┬───────────┬──────────┬──────┬────────────┐
│ 0x02 │ 4B NodeID │ key      │ 0x00 │ value      │
│ 1B   │ uint32 BE │ UTF-8    │ null │ UTF-8      │
└──────┴───────────┴──────────┴──────┴────────────┘
```

- **Key/Value separator**: Null byte (`0x00`).
- Key and value are raw UTF-8.
- **Total size**: 6 + len(key) + len(value) bytes.

##### `0x03` OpRemoveAttr

Remove an HTML attribute from a DOM node.

```
┌──────┬───────────┬──────────┐
│ 0x03 │ 4B NodeID │ key      │
│ 1B   │ uint32 BE │ UTF-8    │
└──────┴───────────┴──────────┘
```

- **Total size**: 5 + len(key) bytes.

##### `0x04` OpInsertNode

Insert a new element into the DOM tree.

```
┌──────┬───────────┬────────────┬────────┬──────────┬────────────┬───────────┐
│ 0x04 │ 4B Parent │ 4B Sibling │ 1B Tag │ tag      │ 2B Attr    │ attrs...  │
│ 1B   │ uint32 BE │ uint32 BE  │ Len    │ UTF-8    │ Count BE   │           │
└──────┴───────────┴────────────┴────────┴──────────┴────────────┴───────────┘
```

**Attribute encoding** (repeated `AttrCount` times):

```
┌──────────┬───────────┬──────────┬─────────────┐
│ 2B KLen  │ key bytes │ 2B VLen  │ value bytes │
│ uint16BE │ UTF-8     │ uint16BE │ UTF-8       │
└──────────┴───────────┴──────────┴─────────────┘
```

- **SiblingID = 0**: Append as last child of parent.
- **ParentID = 0**: Append to document root (`#cbs-root`).
- **Tag length**: Single byte, max 255 characters.

##### `0x05` OpRemoveNode

Remove a DOM node and all its children.

```
┌──────┬───────────┐
│ 0x05 │ 4B NodeID │
│ 1B   │ uint32 BE │
└──────┴───────────┘
```

- **Total size**: 5 bytes (fixed).

##### `0x06` OpReplaceText

Targeted text replacement within a text node (surgical substring update).

```
┌──────┬───────────┬───────────┬───────────┬──────────────────┐
│ 0x06 │ 4B NodeID │ 4B Offset │ 4B Length │ UTF-8 replacement│
│ 1B   │ uint32 BE │ uint32 BE │ uint32 BE │ variable length  │
└──────┴───────────┴───────────┴───────────┴──────────────────┘
```

- **Offset**: Byte position within the existing text to begin replacement.
- **Length**: Number of bytes to replace.
- **Total size**: 13 + len(replacement) bytes.

##### `0x07` OpSetStyle

Set an inline CSS property on a node.

```
┌──────┬───────────┬──────────────┬──────┬──────────────┐
│ 0x07 │ 4B NodeID │ property     │ 0x00 │ value        │
│ 1B   │ uint32 BE │ UTF-8        │ null │ UTF-8        │
└──────┴───────────┴──────────────┴──────┴──────────────┘
```

- Same null-separator pattern as `OpSetAttr`.
- **Total size**: 6 + len(property) + len(value) bytes.

##### `0x08` OpPushHistory

Trigger `history.pushState` for client-side URL updates.

```
┌──────┬─────────────┐
│ 0x08 │ UTF-8 path  │
│ 1B   │ variable    │
└──────┴─────────────┘
```

- No NodeID — this is a global browser operation.
- **Total size**: 1 + len(path) bytes.

##### `0x09` OpBatch

Wrap multiple opcodes into a single atomic frame.

```
┌──────┬──────────┬───────────────────────┐
│ 0x09 │ 4B Count │ nested opcode frames  │
│ 1B   │ uint32BE │ variable length       │
└──────┴──────────┴───────────────────────┘
```

- Nested opcodes are concatenated without additional framing.
- The client processes all nested ops before yielding to the browser render loop, ensuring atomic visual updates.

#### Client → Server Opcodes (0x10 – 0x1F)

##### `0x10` OpClientIntent

Relay a user interaction event to the server.

```
┌──────┬───────────┬───────────┬──────────────┐
│ 0x10 │ 4B NodeID │ 1B Event  │ payload      │
│ 1B   │ uint32 BE │ Type      │ variable     │
└──────┴───────────┴───────────┴──────────────┘
```

- **Total size**: 6 + len(payload) bytes.

##### `0x11` OpClientNav

Signal client-side navigation (popstate or link click).

```
┌──────┬─────────────┐
│ 0x11 │ UTF-8 path  │
│ 1B   │ variable    │
└──────┴─────────────┘
```

- **Total size**: 1 + len(path) bytes.

### Event Type Constants

| Constant | Hex | Description |
|---|---|---|
| `EventClick` | `0x01` | Mouse click or tap |
| `EventInput` | `0x02` | Input field value change |
| `EventSubmit` | `0x03` | Form submission |
| `EventFocus` | `0x04` | Element received focus |
| `EventBlur` | `0x05` | Element lost focus |
| `EventKeyDown` | `0x06` | Key pressed |
| `EventKeyUp` | `0x07` | Key released |
| `EventMouseEnter` | `0x08` | Mouse entered element |
| `EventMouseLeave` | `0x09` | Mouse left element |

### Decoder Errors

| Error | Value | Condition |
|---|---|---|
| `ErrShortRead` | `"cbs: unexpected end of message"` | Frame truncated — not enough bytes for the opcode's fixed prefix |
| `ErrUnknownOp` | `"cbs: unknown opcode"` | Opcode byte not in the 0x01–0x11 range |
| `ErrInvalidFrame` | `"cbs: invalid frame structure"` | Missing null separator in key-value opcodes |

### Wire Format Examples

**Update text "OK" on node 7:**
```
Hex: 01 00 00 00 07 4F 4B
       │  └──────────┘  └─┘
       │     NodeID=7   "OK"
     OpUpdateText
```

**Set class="active" on node 42:**
```
Hex: 02 00 00 00 2A 63 6C 61 73 73 00 61 63 74 69 76 65
       │  └──────────┘  └──────────┘ │  └──────────────┘
       │    NodeID=42     "class"  null    "active"
     OpSetAttr
```

**Remove node 99:**
```
Hex: 05 00 00 00 63
       │  └──────────┘
       │    NodeID=99
     OpRemoveNode
```

---

## 4. Reactive State System

### Package: `pkg/dom`

The reactive state system is built on `Signal[T]` — a generic, thread-safe, observable value container that drives automatic DOM updates.

### Signal[T]

```go
type Signal[T comparable] struct {
    id        SignalID
    mu        sync.RWMutex
    value     T
    dirty     atomic.Bool
    observers []func(T)
}
```

**Type constraint**: `T` must be `comparable` so that equality checks can gate no-op updates. This prevents unnecessary observer notifications and diff cycles.

#### Creation

```go
count := dom.NewSignal(0)          // Signal[int]
name := dom.NewSignal("untitled")  // Signal[string]
active := dom.NewSignal(false)     // Signal[bool]
```

`NewSignal` assigns a globally unique `SignalID` via an atomic counter. IDs are monotonically increasing and never reused.

#### Read

```go
v := count.Get()  // Acquires RLock, copies value, releases
```

Thread-safe. Multiple goroutines may read concurrently.

#### Write

```go
count.Set(42)  // Acquires write lock, compares, notifies observers
```

**Semantics:**
1. Acquires exclusive lock.
2. Compares new value to current via `==`.
3. If equal → unlock and return (no-op). No dirty flag, no observers.
4. If different → store new value, set `dirty = true`, snapshot observer list, unlock.
5. Invoke each observer with the new value.

Observer invocation happens **outside the lock** to prevent deadlocks in observer chains.

#### Functional Update

```go
count.Update(func(v int) int { return v + 1 })
```

Same semantics as `Set` but reads the current value atomically. Avoids the read-then-write race.

#### Dirty Tracking

```go
if count.IsDirty() {
    // Emit delta
    count.Flush()
}
```

- `IsDirty()`: Returns `true` if the value changed since the last `Flush()`.
- `Flush()`: Clears the dirty flag. Called by the engine after binary deltas are emitted.

The dirty flag is an `atomic.Bool` — checking it requires no lock acquisition.

#### Observer Pattern

```go
unsub := count.Observe(func(v int) {
    fmt.Println("count changed to", v)
})

// Later:
unsub()  // Nils out the observer slot
```

- Observers are stored in a slice. Unsubscription nils the slot rather than shifting to avoid index invalidation.
- The observer list is **snapshot-copied** before invocation to prevent concurrent modification.
- Nil observers are skipped during notification.

#### Threading Model

| Operation | Lock Type | Blocks Writers | Blocks Readers |
|---|---|---|---|
| `Get()` | `RLock` | No | No |
| `Set()` | `Lock` | Yes | Yes |
| `Update()` | `Lock` | Yes | Yes |
| `IsDirty()` | Atomic | No | No |
| `Flush()` | Atomic | No | No |
| `Observe()` | `Lock` | Yes | Yes |

### Computed Signals

```go
func Computed[T comparable, U comparable](source *Signal[T], derive func(T) U) *Signal[U]
```

Creates a derived signal that automatically recomputes when the source changes.

```go
count := dom.NewSignal(3)
doubled := dom.Computed(count, func(v int) int { return v * 2 })

doubled.Get()  // 6
count.Set(10)
doubled.Get()  // 20
```

**Implementation**: `Computed` creates a new `Signal[U]`, computes the initial value, then registers an observer on the source that calls `derived.Set(derive(v))` on each change. The derived signal's own equality check gates further propagation.

### Signal Lifecycle

```
NewSignal(initial)
    │
    ▼
 [Clean: dirty=false]
    │
    │ Set(newValue) where newValue != current
    ▼
 [Dirty: dirty=true, observers notified]
    │
    │ Engine emits binary delta
    │
    │ Flush()
    ▼
 [Clean: dirty=false]
    │
    (cycle repeats)
```

---

## 5. Virtual DOM & Component Model

### Package: `pkg/dom`

### Node Structure

```go
type Node struct {
    ID       NodeID            // Globally unique uint32
    Type     NodeType          // ElementNode (1) or TextNode (2)
    Tag      string            // HTML tag name (elements only)
    Attrs    map[string]string // HTML attributes
    Styles   map[string]string // Inline CSS properties
    Text     string            // Text content (text nodes only)
    Children []*Node           // Ordered child nodes
    Parent   *Node             // Back-pointer for tree traversal
    Handlers map[byte]func([]byte)  // Event handlers keyed by EventType
}
```

**NodeID allocation**: Global atomic counter (`atomic.Uint32`). IDs are monotonically increasing and unique within the process lifetime.

**Node types**:
- `ElementNode` (1): Has a tag, attributes, styles, children, and handlers.
- `TextNode` (2): Has text content only. No children, no handlers.

### Element Builder API

CBS uses the **functional options pattern** for element construction. Every HTML element has a corresponding Go function:

```go
// Create a div with attributes and children
dom.Div(
    dom.Class("container"),
    dom.ID("main"),
    dom.Style("background-color", "#fff"),
    dom.Children(
        dom.H1(dom.Children(dom.Text("Hello"))),
        dom.P(dom.Children(dom.Text("World"))),
    ),
)
```

#### Available Element Constructors

| Function | HTML Tag | Function | HTML Tag |
|---|---|---|---|
| `Div` | `<div>` | `Nav` | `<nav>` |
| `Span` | `<span>` | `Header` | `<header>` |
| `P` | `<p>` | `Footer` | `<footer>` |
| `H1` | `<h1>` | `Main` | `<main>` |
| `H2` | `<h2>` | `Section` | `<section>` |
| `H3` | `<h3>` | `Article` | `<article>` |
| `Button` | `<button>` | `Img` | `<img>` |
| `Input` | `<input>` | `Label` | `<label>` |
| `Form` | `<form>` | `Table` | `<table>` |
| `A` | `<a>` | `Tr` | `<tr>` |
| `Ul` | `<ul>` | `Td` | `<td>` |
| `Li` | `<li>` | `Th` | `<th>` |

For non-standard or custom elements:

```go
dom.El("custom-element", dom.Attr("data-id", "123"))
```

#### Option Functions

| Option | Signature | Purpose |
|---|---|---|
| `Attr` | `Attr(key, value string) Option` | Set any HTML attribute |
| `ID` | `ID(id string) Option` | Set the `id` attribute |
| `Class` | `Class(cls string) Option` | Set the `class` attribute |
| `Style` | `Style(property, value string) Option` | Set an inline CSS property |
| `OnClick` | `OnClick(fn func([]byte)) Option` | Register click handler |
| `OnInput` | `OnInput(fn func([]byte)) Option` | Register input handler |
| `OnSubmit` | `OnSubmit(fn func([]byte)) Option` | Register submit handler |
| `On` | `On(eventType byte, fn func([]byte)) Option` | Register any event handler |
| `Children` | `Children(children ...*Node) Option` | Append child nodes |

#### Text Nodes

Static text:
```go
dom.Text("Hello, world")
```

Signal-bound text (auto-updating):
```go
count := dom.NewSignal(0)
dom.TextF(count, func(v int) string {
    return fmt.Sprintf("Count: %d", v)
})
```

`TextF` registers an observer on the signal that mutates the node's `Text` field in-place whenever the signal value changes.

### Component Contract

A CBS component is a function with the signature:

```go
type Component func(s *engine.Session) *dom.Node
```

- Receives the session for context (e.g., session-scoped state, cancellation).
- Returns the root `*dom.Node` tree.
- Called once per session connection (on mount).

**Example:**

```go
func counterApp(s *engine.Session) *dom.Node {
    count := dom.NewSignal(0)

    return dom.Div(
        dom.Children(
            dom.TextF(count, func(v int) string {
                return fmt.Sprintf("Count: %d", v)
            }),
            dom.Button(
                dom.Children(dom.Text("+")),
                dom.OnClick(func(_ []byte) {
                    count.Update(func(v int) int { return v + 1 })
                }),
            ),
        ),
    )
}
```

### Diff Algorithm

**Package**: `pkg/dom` — `Diff(buf *protocol.Buffer, old, next *Node)`

The differ compares two node trees and emits the minimal set of binary opcodes to reconcile them.

#### Algorithm

```
Diff(old, next):
    if both nil       → return
    if old nil         → emitInsert(next)   // new subtree
    if next nil        → EncodeRemoveNode(old.ID)
    if both TextNode   → if text differs, EncodeUpdateText
    for each attr in next:
        if old[attr] != next[attr] → EncodeSetAttr
    for each attr in old not in next:
        → EncodeRemoveAttr
    for each style in next:
        if old[style] != next[style] → EncodeSetStyle
    for i in 0..max(len(old.Children), len(next.Children)):
        Diff(old.Children[i], next.Children[i])  // recursive
```

#### Characteristics

- **O(n)** where n = total node count (no key-based reordering — linear child comparison).
- Attribute diffing is **O(a)** where a = attribute count on a single node.
- Style diffing follows the same pattern.
- Child insertions and removals are detected positionally.

#### Insert Emission

When a node exists in `next` but not in `old`, the differ emits a full `OpInsertNode` for element nodes (with all attributes) and `OpUpdateText` for text nodes, then recurses into children.

---

## 6. Server Engine

### Package: `pkg/engine`

### Server

```go
type Server struct {
    addr      string
    tlsConfig *tls.Config
    component Component
    logger    *slog.Logger
    mu        sync.Mutex
    sessions  map[SessionID]*Session
    wt        *webtransport.Server
}
```

#### Construction

```go
srv := engine.NewServer(
    ":4433",
    tlsConfig,
    counterApp,
    engine.WithLogger(logger),
)
```

| Parameter | Type | Description |
|---|---|---|
| `addr` | `string` | Listen address (e.g., `:4433`) |
| `tlsConfig` | `*tls.Config` | TLS configuration (TLS 1.3 required for HTTP/3) |
| `comp` | `Component` | Root component function |
| `opts` | `...ServerOption` | Optional configuration |

#### Server Options

| Option | Signature | Purpose |
|---|---|---|
| `WithLogger` | `WithLogger(l *slog.Logger) ServerOption` | Set structured logger |

#### Lifecycle

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

go func() {
    <-ctx.Done()
    srv.Shutdown()
}()

srv.ListenAndServe(ctx)
```

`ListenAndServe` starts an HTTP/3 server with two endpoints:

| Path | Purpose |
|---|---|
| `/cbs` | WebTransport upgrade endpoint — spawns a session per connection |
| `/` | Serves the bootstrap HTML shell that loads the WASM client |

#### Bootstrap HTML Shell

```html
<!DOCTYPE html>
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
</html>
```

The `#cbs-root` div is the mount point. The WASM client attaches here.

#### Connection Handling

1. Browser requests `/cbs` with WebTransport upgrade.
2. Server calls `webtransport.Server.Upgrade()`.
3. A new `Session` is created with a `wtWriter` adapter.
4. The component is mounted: `sess.Mount(component)` renders the tree and sends initial opcodes.
5. A goroutine accepts bidirectional streams for client intents.
6. On disconnect, the session is removed from the map and closed.

#### Writer Interface

```go
type Writer interface {
    WriteMessage(data []byte) error
    Close() error
}
```

The `wtWriter` implementation opens a unidirectional stream per message:
1. `OpenUniStream()` — creates a new QUIC uni-stream.
2. `Write(data)` — writes the opcode bytes.
3. `Close()` — closes the stream (not the session).

### Session

```go
type Session struct {
    ID     SessionID
    ctx    context.Context
    cancel context.CancelFunc
    mu     sync.Mutex
    root   *dom.Node
    writer Writer
    logger *slog.Logger
}
```

#### Mount

`Mount(comp Component) error`:
1. Calls the component function to build the virtual DOM tree.
2. Acquires a pooled `Buffer`.
3. Serializes the entire tree as `OpInsertNode` + `OpUpdateText` opcodes via `emitFullTree`.
4. Writes the buffer to the transport.

#### HandleIntent

`HandleIntent(data []byte) error`:
1. Decodes the binary message.
2. Validates it's an `OpClientIntent`.
3. DFS-searches the virtual DOM tree for the target `NodeID`.
4. Looks up the handler for the event type.
5. Invokes the handler with the event payload.

The handler typically mutates signal values, which triggers observers, which update DOM node text/attributes, which will be detected by the next diff cycle.

#### Session IDs

`SessionID` is a `uint64` assigned via a mutex-protected counter. Sessions are stored in `Server.sessions` map for lifecycle management.

### Broadcast (Pub/Sub)

```go
type Broadcast struct {
    mu          sync.RWMutex
    subscribers map[string][]chan []byte
}
```

#### API

| Method | Signature | Description |
|---|---|---|
| `NewBroadcast` | `NewBroadcast() *Broadcast` | Create a new broadcast hub |
| `Subscribe` | `Subscribe(topic string) chan []byte` | Subscribe to a topic, returns buffered channel (cap 64) |
| `Publish` | `Publish(topic string, data []byte)` | Send data to all topic subscribers |
| `Unsubscribe` | `Unsubscribe(topic string, ch chan []byte)` | Remove and close a subscription channel |

#### Backpressure Policy

`Publish` uses a non-blocking channel send. If a subscriber's channel is full (64 messages buffered), the message is **dropped** for that subscriber. This prevents slow consumers from blocking the publisher or other subscribers.

#### Use Cases

- Live notifications across sessions
- Collaborative editing (broadcast mutations to all viewers)
- Real-time dashboards (push data updates to all connected sessions)

---

## 7. WASM Client Runtime

### Package: `pkg/wasm`

Build constraint: `//go:build js && wasm`

The WASM client is a minimal Go binary compiled to WebAssembly. It has one job: receive binary opcodes and apply them to the browser DOM via `syscall/js`.

### Initialization

```go
func Start() {
    root = document.Call("getElementById", "cbs-root")
    js.Global().Set("__cbs_patch", js.FuncOf(patchFromJS))
    select {}  // Keep alive
}
```

1. Locates the `#cbs-root` element.
2. Exports `__cbs_patch` to the JavaScript global scope.
3. Blocks forever (WASM modules exit when `main()` returns).

### JS Interop

The bootstrap JavaScript (loaded as `/cbs.js`) establishes the WebTransport connection and calls `__cbs_patch(uint8Array)` when binary frames arrive.

```
JavaScript                    Go (WASM)
    │                              │
    │  WebTransport.receive()      │
    │  ──────────────────────►     │
    │  __cbs_patch(bytes)          │
    │                    patchFromJS()
    │                    js.CopyBytesToGo()
    │                    applyOpcodes()
    │                              │
    │                    DOM mutations via syscall/js
    │  ◄──────────────────────     │
    │  Browser renders             │
```

### Node Registry

```go
var nodes map[uint32]js.Value
```

The client maintains a map of `nodeID → js.Value` (DOM element references). When `OpInsertNode` creates an element, it's registered here. When `OpRemoveNode` removes one, the entry is deleted.

### Opcode Interpreter

The `applyOpcodes(data []byte)` function is a sequential interpreter that processes opcodes from a byte stream:

| Opcode | Client Action |
|---|---|
| `0x01` OpUpdateText | `node.Set("textContent", text)` |
| `0x02` OpSetAttr | `node.Call("setAttribute", key, value)` |
| `0x03` OpRemoveAttr | `node.Call("removeAttribute", key)` |
| `0x04` OpInsertNode | `document.Call("createElement", tag)` + set attrs + `parent.Call("appendChild", el)` |
| `0x05` OpRemoveNode | `parent.Call("removeChild", node)` + delete from registry |
| `0x07` OpSetStyle | `node.Get("style").Call("setProperty", prop, val)` |
| `0x08` OpPushHistory | `history.Call("pushState", nil, "", path)` |

### XSS Immunity

The WASM client never uses `innerHTML`, `outerHTML`, or `document.write`. Every DOM mutation is performed via:

- `textContent` (automatically escapes HTML entities)
- `setAttribute` (sets attribute values, not HTML)
- `createElement` + `appendChild` (structural, not string-based)

This architecture is **structurally immune to XSS** — there is no code path where user-provided data can be interpreted as HTML or script.

---

## 8. Security Architecture

### Zero-Trust Model

CBS operates on the principle that the client is untrusted:

1. **Server owns all state** — The client has no local state to compromise. No `localStorage` tokens, no client-side business logic.
2. **Client sends intents, not commands** — The client can only say "the user clicked node X" or "the user typed Y." The server decides what happens.
3. **No `eval`, no `innerHTML`** — The WASM client only calls safe DOM APIs.

### Transport Security

| Requirement | Implementation |
|---|---|
| **TLS 1.3 mandatory** | `tls.Config{MinVersion: tls.VersionTLS13}` |
| **HTTP/3 + QUIC** | `quic-go/quic-go` + `quic-go/webtransport-go` |
| **0-RTT not used** | Standard QUIC handshake (prevents replay attacks) |
| **Certificate pinning** | Supported via `tls.Config.VerifyPeerCertificate` |

### WebAuthn Integration

The WASM client includes native WebAuthn (passkey) support:

```go
func RequestPasskey(rpID string, challenge []byte) (js.Value, error)
```

- Builds `PublicKeyCredentialRequestOptions` via `syscall/js`.
- Calls `navigator.credentials.get()` for passwordless authentication.
- Returns the browser's `Promise<PublicKeyCredential>`.

**Flow:**

```
Server                          Browser (WASM)
   │                                  │
   │  Send challenge bytes            │
   │  ─────────────────────────►      │
   │                                  │
   │                   RequestPasskey(rpID, challenge)
   │                   navigator.credentials.get()
   │                                  │
   │                   User touches authenticator
   │                                  │
   │  ◄─────────────────────────      │
   │  Receive signed assertion        │
   │                                  │
   │  Verify signature (server-side)  │
```

### XSS Prevention

| Attack Vector | CBS Mitigation |
|---|---|
| `innerHTML` injection | Not used — all mutations via `textContent`, `setAttribute`, `createElement` |
| Script injection via attributes | Attributes are set via `setAttribute`, not string concatenation |
| Event handler injection | Handlers are Go functions registered server-side; clients send event type bytes, not code |
| URL injection | `OpPushHistory` paths are server-generated |

### Threat Model

| Threat | Mitigation |
|---|---|
| Binary protocol fuzzing | Decoder validates all lengths before reads; returns `ErrShortRead` on truncation |
| Session hijacking | WebTransport sessions are bound to QUIC connections with TLS 1.3 |
| Replay attacks | Standard QUIC handshake (not 0-RTT) |
| Denial of service (intent flood) | Backpressure: handlers run synchronously per session; the server controls processing rate |
| Slow subscriber backpressure | Broadcast drops messages to full channels (cap 64) |

---

## 9. Styling System

### Package: `pkg/style`

### Design Philosophy

CBS provides a **type-safe, compile-time-checked CSS system**. Utility class names are Go constants — the compiler catches typos, and unused classes are eliminated by the CSS generator.

### Class Type

```go
type Class string
```

Every CSS utility is a typed constant:

```go
const (
    Flex          Class = "flex"
    FlexCol       Class = "flex-col"
    ItemsCenter   Class = "items-center"
    BgBlue500     Class = "bg-blue-500"
    TextWhite     Class = "text-white"
    // ... 55 total constants
)
```

### Available Classes

#### Layout

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `Flex` | `flex` | `display:flex` |
| `FlexCol` | `flex-col` | `flex-direction:column` |
| `FlexRow` | `flex-row` | `flex-direction:row` |
| `FlexWrap` | `flex-wrap` | `flex-wrap:wrap` |
| `ItemsCenter` | `items-center` | `align-items:center` |
| `JustifyCenter` | `justify-center` | `justify-content:center` |
| `JustifyBetween` | `justify-between` | `justify-content:space-between` |
| `Gap1` | `gap-1` | `gap:0.25rem` |
| `Gap2` | `gap-2` | `gap:0.5rem` |
| `Gap4` | `gap-4` | `gap:1rem` |
| `Gap8` | `gap-8` | `gap:2rem` |

#### Spacing

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `P1` | `p-1` | `padding:0.25rem` |
| `P2` | `p-2` | `padding:0.5rem` |
| `P4` | `p-4` | `padding:1rem` |
| `P8` | `p-8` | `padding:2rem` |
| `M1` | `m-1` | `margin:0.25rem` |
| `M2` | `m-2` | `margin:0.5rem` |
| `M4` | `m-4` | `margin:1rem` |
| `M8` | `m-8` | `margin:2rem` |
| `Mx2` | `mx-2` | `margin-left:0.5rem;margin-right:0.5rem` |
| `My2` | `my-2` | `margin-top:0.5rem;margin-bottom:0.5rem` |
| `Px4` | `px-4` | `padding-left:1rem;padding-right:1rem` |
| `Py2` | `py-2` | `padding-top:0.5rem;padding-bottom:0.5rem` |

#### Typography

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `TextSm` | `text-sm` | `font-size:0.875rem;line-height:1.25rem` |
| `TextBase` | `text-base` | `font-size:1rem;line-height:1.5rem` |
| `TextLg` | `text-lg` | `font-size:1.125rem;line-height:1.75rem` |
| `TextXl` | `text-xl` | `font-size:1.25rem;line-height:1.75rem` |
| `Text2Xl` | `text-2xl` | `font-size:1.5rem;line-height:2rem` |
| `Text3Xl` | `text-3xl` | `font-size:1.875rem;line-height:2.25rem` |
| `FontBold` | `font-bold` | `font-weight:700` |
| `FontMedium` | `font-medium` | `font-weight:500` |
| `TextCenter` | `text-center` | `text-align:center` |

#### Colors

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `BgWhite` | `bg-white` | `background-color:#ffffff` |
| `BgGray100` | `bg-gray-100` | `background-color:#f3f4f6` |
| `BgGray800` | `bg-gray-800` | `background-color:#1f2937` |
| `BgBlue500` | `bg-blue-500` | `background-color:#3b82f6` |
| `BgBlue600` | `bg-blue-600` | `background-color:#2563eb` |
| `BgGreen500` | `bg-green-500` | `background-color:#22c55e` |
| `BgRed500` | `bg-red-500` | `background-color:#ef4444` |
| `TextWhite` | `text-white` | `color:#ffffff` |
| `TextGray700` | `text-gray-700` | `color:#374151` |
| `TextGray900` | `text-gray-900` | `color:#111827` |
| `TextBlue500` | `text-blue-500` | `color:#3b82f6` |

#### Borders & Rounding

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `Rounded` | `rounded` | `border-radius:0.25rem` |
| `RoundedMd` | `rounded-md` | `border-radius:0.375rem` |
| `RoundedLg` | `rounded-lg` | `border-radius:0.5rem` |
| `RoundedFull` | `rounded-full` | `border-radius:9999px` |
| `Border` | `border` | `border-width:1px` |
| `BorderGray300` | `border-gray-300` | `border-color:#d1d5db` |
| `Shadow` | `shadow` | `box-shadow:0 1px 3px rgba(0,0,0,0.1)` |
| `ShadowMd` | `shadow-md` | `box-shadow:0 4px 6px rgba(0,0,0,0.1)` |
| `ShadowLg` | `shadow-lg` | `box-shadow:0 10px 15px rgba(0,0,0,0.1)` |

#### Sizing

| Constant | CSS Class | CSS Rule |
|---|---|---|
| `WFull` | `w-full` | `width:100%` |
| `HFull` | `h-full` | `height:100%` |
| `WScreen` | `w-screen` | `width:100vw` |
| `HScreen` | `h-screen` | `height:100vh` |
| `MaxWLg` | `max-w-lg` | `max-width:32rem` |
| `MaxWXl` | `max-w-xl` | `max-width:36rem` |
| `MaxW2Xl` | `max-w-2xl` | `max-width:42rem` |

### Class Composition

```go
dom.Class(style.Classes(style.Flex, style.FlexCol, style.ItemsCenter, style.Gap4))
// → class="flex flex-col items-center gap-4"
```

`Classes` joins multiple typed classes with spaces. The Go compiler ensures every argument is a valid `style.Class` constant.

### CSS Generation

#### Dead-Code Eliminating Generator

```go
css := style.Generate([]style.Class{
    style.Flex,
    style.BgBlue500,
    style.TextWhite,
})
```

Output:
```css
/* CBS Generated Stylesheet - Zero Dead Code */
.flex{display:flex}
.bg-blue-500{background-color:#3b82f6}
.text-white{color:#ffffff}
```

Only classes passed to `Generate` are emitted. If your application uses 5 classes, you get 5 CSS rules — not a 200KB utility framework.

#### Full Generation

```go
css := style.GenerateAll()
```

Emits all registered classes. Useful for development/debugging.

---

## 10. CLI Toolchain

### Binary: `cmd/cbs`

```
CBS - Continuous Binary Synchronization

Usage:
  cbs <command>

Commands:
  build    Compile the WASM client module
  serve    Start the development server
  version  Print version information
```

### `cbs build`

Compiles the WASM client:

```bash
GOOS=js GOARCH=wasm go build -o dist/cbs.wasm ./pkg/wasm
```

Creates the `dist/` directory if it doesn't exist.

### `cbs serve`

Starts the development server with hot reload (planned — currently a placeholder).

### `cbs version`

Prints the current CBS version:

```
cbs v0.1.0
```

### Build Pipeline

```
Source (.go)                    Output
    │                              │
    ├── Server binary ◄──── go build ./cmd/server
    │                              │
    └── WASM client  ◄──── GOOS=js GOARCH=wasm go build ./pkg/wasm
                                   │
                              dist/cbs.wasm
```

The server binary and WASM client are built from the same Go module. They share the `protocol` package for opcode constants, ensuring client and server are always in sync.

---

## 11. Offline & Resilience

### Package: `pkg/wasm`

### OfflineQueue

When the WebTransport connection drops, user interactions are buffered locally rather than lost.

```go
type OfflineQueue struct {
    mu      sync.Mutex
    queue   [][]byte
    maxSize int
}
```

#### API

| Method | Signature | Description |
|---|---|---|
| `NewOfflineQueue` | `NewOfflineQueue(maxSize int) *OfflineQueue` | Create queue with capacity limit |
| `Enqueue` | `Enqueue(data []byte) bool` | Buffer a binary message; returns `false` if full |
| `Flush` | `Flush() [][]byte` | Return all buffered messages and clear the queue |
| `Len` | `Len() int` | Number of queued messages |

#### Queue Semantics

- **Bounded**: `maxSize` prevents unbounded memory growth during extended outages.
- **Copy-on-enqueue**: Input data is copied to prevent external mutation of queued messages.
- **FIFO**: Messages are flushed in the order they were enqueued.
- **Thread-safe**: All operations are mutex-protected.
- **Pre-allocated**: Initial capacity of 64 slots to minimize early allocations.

#### Reconnection Protocol

```
Connection drops
       │
       ▼
  ┌──────────────┐
  │ User actions  │──► OfflineQueue.Enqueue(binary)
  │ continue      │    (returns false if queue full)
  └──────────────┘
       │
       │ Connection restored
       ▼
  ┌──────────────┐
  │ Flush()      │──► Returns [][]byte
  └──────┬───────┘
         │
         ▼
  ┌──────────────┐
  │ Send each    │──► WebTransport stream
  │ message      │
  └──────────────┘
```

#### Design Considerations

- **No deduplication**: Flushed messages are sent as-is. The server is responsible for idempotency if needed.
- **No persistence**: The queue lives in WASM memory. A full page refresh loses queued intents.
- **Bounded loss**: When the queue is full, new intents are dropped (returns `false`). The application can notify the user.

---

## 12. Developer API Reference

### Package `protocol` — `github.com/bytewiredev/bytewire/pkg/protocol`

Binary instruction set encoder, decoder, and opcode constants.

#### Constants

```go
// Server → Client
const OpUpdateText   byte = 0x01
const OpSetAttr      byte = 0x02
const OpRemoveAttr   byte = 0x03
const OpInsertNode   byte = 0x04
const OpRemoveNode   byte = 0x05
const OpReplaceText  byte = 0x06
const OpSetStyle     byte = 0x07
const OpPushHistory  byte = 0x08
const OpBatch        byte = 0x09

// Client → Server
const OpClientIntent byte = 0x10
const OpClientNav    byte = 0x11

// Event Types
const EventClick      byte = 0x01
const EventInput      byte = 0x02
const EventSubmit     byte = 0x03
const EventFocus      byte = 0x04
const EventBlur       byte = 0x05
const EventKeyDown    byte = 0x06
const EventKeyUp      byte = 0x07
const EventMouseEnter byte = 0x08
const EventMouseLeave byte = 0x09
```

#### Errors

```go
var ErrShortRead    = errors.New("cbs: unexpected end of message")
var ErrUnknownOp    = errors.New("cbs: unknown opcode")
var ErrInvalidFrame = errors.New("cbs: invalid frame structure")
```

#### Types

```go
type Buffer struct { /* pooled binary encoder */ }

func AcquireBuffer() *Buffer
func (b *Buffer) Release()
func (b *Buffer) Bytes() []byte
func (b *Buffer) Len() int
func (b *Buffer) WriteTo(w io.Writer) (int64, error)
func (b *Buffer) Reset()
func (b *Buffer) EncodeUpdateText(nodeID uint32, text string)
func (b *Buffer) EncodeSetAttr(nodeID uint32, key, value string)
func (b *Buffer) EncodeRemoveAttr(nodeID uint32, key string)
func (b *Buffer) EncodeInsertNode(parentID, siblingID uint32, tag string, attrs map[string]string)
func (b *Buffer) EncodeRemoveNode(nodeID uint32)
func (b *Buffer) EncodeSetStyle(nodeID uint32, property, value string)
func (b *Buffer) EncodePushHistory(path string)
func (b *Buffer) EncodeClientIntent(nodeID uint32, eventType byte, payload []byte)
func (b *Buffer) EncodeClientNav(path string)
```

```go
type Message struct {
    Op        byte
    NodeID    uint32
    ParentID  uint32
    SiblingID uint32
    Tag       string
    Attrs     [][2]string
    Key       string
    Value     string
    Text      string
    EventType byte
    Payload   []byte
}

func Decode(data []byte) (Message, int, error)
```

---

### Package `dom` — `github.com/bytewiredev/bytewire/pkg/dom`

Virtual DOM tree, reactive signals, element builders, and tree diffing.

#### Signal Types

```go
type SignalID uint64

type Signal[T comparable] struct { /* ... */ }

func NewSignal[T comparable](initial T) *Signal[T]
func (s *Signal[T]) Get() T
func (s *Signal[T]) Set(v T)
func (s *Signal[T]) Update(fn func(T) T)
func (s *Signal[T]) IsDirty() bool
func (s *Signal[T]) Flush()
func (s *Signal[T]) ID() SignalID
func (s *Signal[T]) Observe(fn func(T)) func()

func Computed[T comparable, U comparable](source *Signal[T], derive func(T) U) *Signal[U]
```

#### Node Types

```go
type NodeID uint32
type NodeType byte

const ElementNode NodeType = 1
const TextNode    NodeType = 2

type Node struct {
    ID       NodeID
    Type     NodeType
    Tag      string
    Attrs    map[string]string
    Styles   map[string]string
    Text     string
    Children []*Node
    Parent   *Node
    Handlers map[byte]func([]byte)
}

func (n *Node) AppendChild(child *Node) *Node
```

#### Option Type & Builders

```go
type Option func(*Node)

func Attr(key, value string) Option
func ID(id string) Option
func Class(cls string) Option
func Style(property, value string) Option
func OnClick(fn func([]byte)) Option
func OnInput(fn func([]byte)) Option
func OnSubmit(fn func([]byte)) Option
func On(eventType byte, fn func([]byte)) Option
func Children(children ...*Node) Option
```

#### Element Constructors

```go
func Div(opts ...Option) *Node
func Span(opts ...Option) *Node
func P(opts ...Option) *Node
func H1(opts ...Option) *Node
func H2(opts ...Option) *Node
func H3(opts ...Option) *Node
func Button(opts ...Option) *Node
func Input(opts ...Option) *Node
func Form(opts ...Option) *Node
func A(opts ...Option) *Node
func Ul(opts ...Option) *Node
func Li(opts ...Option) *Node
func Nav(opts ...Option) *Node
func Header(opts ...Option) *Node
func Footer(opts ...Option) *Node
func Main(opts ...Option) *Node
func Section(opts ...Option) *Node
func Article(opts ...Option) *Node
func Img(opts ...Option) *Node
func Label(opts ...Option) *Node
func Table(opts ...Option) *Node
func Tr(opts ...Option) *Node
func Td(opts ...Option) *Node
func Th(opts ...Option) *Node
func El(tag string, opts ...Option) *Node
func Text(content string) *Node
func TextF[T comparable](s *Signal[T], format func(T) string) *Node
```

#### Diff

```go
func Diff(buf *protocol.Buffer, old, next *Node)
```

---

### Package `engine` — `github.com/bytewiredev/bytewire/pkg/engine`

WebTransport server, session management, and binary stream multiplexing.

#### Types

```go
type SessionID uint64

type Component func(s *Session) *dom.Node

type Writer interface {
    WriteMessage(data []byte) error
    Close() error
}

type ServerOption func(*Server)

type Server struct { /* ... */ }

func NewServer(addr string, tlsConfig *tls.Config, comp Component, opts ...ServerOption) *Server
func (s *Server) ListenAndServe(ctx context.Context) error
func (s *Server) Shutdown() error

func WithLogger(l *slog.Logger) ServerOption

type Session struct {
    ID SessionID
    // ...
}

func NewSession(ctx context.Context, w Writer, logger *slog.Logger) *Session
func (s *Session) Mount(comp Component) error
func (s *Session) HandleIntent(data []byte) error
func (s *Session) Close()
func (s *Session) Context() context.Context
```

#### Broadcast

```go
type Broadcast struct { /* ... */ }

func NewBroadcast() *Broadcast
func (b *Broadcast) Subscribe(topic string) chan []byte
func (b *Broadcast) Publish(topic string, data []byte)
func (b *Broadcast) Unsubscribe(topic string, ch chan []byte)
```

---

### Package `style` — `github.com/bytewiredev/bytewire/pkg/style`

Type-safe CSS utility classes and dead-code eliminating CSS generator.

```go
type Class string

func Classes(classes ...Class) string
func Generate(used []Class) string
func GenerateAll() string

// 55 typed constants — see Section 9 for full list
```

---

### Package `wasm` — `github.com/bytewiredev/bytewire/pkg/wasm`

Browser WASM client runtime (build constraint: `js && wasm`).

```go
func Start()
func RequestPasskey(rpID string, challenge []byte) (js.Value, error)

type OfflineQueue struct { /* ... */ }

func NewOfflineQueue(maxSize int) *OfflineQueue
func (q *OfflineQueue) Enqueue(data []byte) bool
func (q *OfflineQueue) Flush() [][]byte
func (q *OfflineQueue) Len() int
```

---

## 13. Performance Characteristics

### Binary Protocol vs JSON

| Metric | CBS Binary | JSON Equivalent |
|---|---|---|
| Text update "Count: 42" | **15 bytes** | ~45 bytes (`{"op":"updateText","nodeId":1024,"text":"Count: 42"}`) |
| Set attribute | **6 + key + value** | ~50+ bytes with JSON overhead |
| Remove node | **5 bytes** | ~30 bytes |
| Parse time | O(1) field reads | O(n) JSON tokenization |
| Allocation | 0 (pooled buffers) | 2+ (string interning, map construction) |

### Buffer Pooling

```go
var bufferPool = sync.Pool{
    New: func() any {
        return &Buffer{buf: make([]byte, 0, 256)}
    },
}
```

- **Initial capacity**: 256 bytes (covers most single-opcode frames).
- **Growth**: Standard Go slice growth (amortized O(1) append).
- **Reuse**: `AcquireBuffer()` / `Release()` cycle. Buffers are reset (length zeroed) but capacity is retained.
- **GC pressure**: Near-zero on steady-state — buffers circulate through the pool.

### Benchmark Results (Protocol Package)

From `pkg/protocol/buffer_test.go`:

| Benchmark | Description |
|---|---|
| `BenchmarkEncodeUpdateText` | Encode a text update (reports allocs) |
| `BenchmarkDecodeUpdateText` | Decode a text update (reports allocs) |

Expected characteristics (zero-allocation hot path):

- Encode: **0 allocs/op** (writes to pooled buffer)
- Decode: **1 alloc/op** (string conversion from byte slice for `msg.Text`)

### Signal Performance

From `pkg/dom/signals_test.go`:

| Benchmark | Description |
|---|---|
| `BenchmarkSignalSet` | Set values in a tight loop (reports allocs) |

Expected: **0 allocs/op** for same-type sets (no observer snapshot when value unchanged).

### WASM Client Overhead

- **Binary size**: Minimal — only `syscall/js`, `encoding/binary`, and opcode dispatch logic.
- **DOM operations**: Direct `syscall/js` calls — no virtual DOM diffing on the client.
- **Memory**: Node registry (`map[uint32]js.Value`) grows linearly with DOM size.

### Latency Model

```
User action → Opcode encoding (client)    : < 1μs
Network transit (WebTransport/QUIC)        : ~RTT (typically 1-50ms)
Server decode + handler dispatch           : < 10μs
Signal update + observer notification      : < 1μs
Diff computation                           : O(n) where n = changed subtree
Binary encoding of deltas                  : < 5μs per opcode
Network transit (server → client)          : ~RTT
WASM decode + DOM patch                    : < 1μs per opcode
Browser render                             : ~16ms (next frame)
─────────────────────────────────────────
Total round-trip                           : ~2×RTT + ~16ms render
```

For a local development setup (RTT ≈ 0): **sub-frame latency** (~16ms total).

---

## 14. Roadmap & Future Work

### v0.2.0 — Core Completions

| Feature | Status | Description |
|---|---|---|
| `OpBatch` client support | Planned | Process batched opcodes atomically in WASM interpreter |
| `OpReplaceText` client support | Planned | Surgical text replacement in WASM |
| Signal-driven diff loop | Planned | Automatic diff cycle when signals fire (currently manual) |
| `cbs serve` implementation | Planned | Development server with file watching and hot reload |
| Actual WASM compilation | Planned | `cbs build` currently creates `dist/` but doesn't invoke `go build` |
| Node ID synchronization | Planned | Server-assigned IDs transmitted to WASM client via `OpInsertNode` |

### v0.3.0 — Developer Experience

| Feature | Description |
|---|---|
| Hot reload | File watcher triggers rebuild + reconnect |
| Error overlay | Server errors displayed in browser |
| DevTools integration | Custom browser panel for CBS state inspection |
| Component library | Pre-built UI components (forms, tables, modals) |
| Router package | Declarative server-side routing with `OpPushHistory` |

### v0.4.0 — Production Readiness

| Feature | Description |
|---|---|
| Connection pooling | Reuse QUIC connections across sessions |
| Rate limiting | Per-session intent throttling |
| Metrics export | Prometheus/OpenTelemetry integration |
| Graceful degradation | Fallback to WebSocket when WebTransport unavailable |
| Key-based child reconciliation | O(n) keyed list diffing (like React's `key` prop) |
| Computed style classes | Reactive `Class` binding driven by signals |

### v1.0.0 — Stable Release

| Feature | Description |
|---|---|
| Stable binary protocol | Versioned wire format with backwards compatibility |
| Full WebAuthn flow | Server-side credential verification |
| Offline persistence | IndexedDB-backed queue for page refresh survival |
| SSR fallback | Initial HTML render for SEO/no-JS clients |
| Plugin system | Middleware hooks for server and client lifecycle |

### Known Gaps (Current Implementation)

1. **No automatic diff cycle** — Signal changes mutate the DOM tree but don't automatically trigger diff + send. The engine needs a tick/flush loop.
2. **WASM node ID assignment** — The client assigns synthetic IDs (`len(nodes) + 1`) rather than using server-provided IDs, causing ID mismatches.
3. **Single-frame parsing** — The WASM `OpUpdateText` handler reads until end-of-data, which fails inside batch frames.
4. **No `OpBatch` client handling** — Batch opcode (`0x09`) is not implemented in the WASM interpreter.
5. **No `OpReplaceText` client handling** — Opcode `0x06` is not implemented in the WASM interpreter.
6. **`cbs build` is a stub** — Creates `dist/` directory but does not actually invoke the Go compiler.
7. **`cbs serve` is a stub** — Prints a TODO message.
8. **No WebSocket fallback** — WebTransport requires HTTP/3; older browsers have no path.

---

## 15. Glossary

| Term | Definition |
|---|---|
| **CBS** | Continuous Binary Synchronization — the framework name and protocol philosophy |
| **Opcode** | A single-byte instruction identifier (e.g., `0x01` = UpdateText) that begins every binary frame |
| **Signal** | A reactive state container (`Signal[T]`) that notifies observers when its value changes |
| **Computed Signal** | A derived `Signal[U]` that automatically recomputes from a source `Signal[T]` |
| **Intent** | A client-to-server message representing a user interaction (click, input, navigation) |
| **Session** | A per-user server-side state container bound to a WebTransport connection |
| **Component** | A Go function `func(s *Session) *Node` that returns a virtual DOM tree |
| **Virtual DOM** | The server-side `*Node` tree representing the UI state |
| **Diff** | The process of comparing two node trees and emitting the minimal set of binary opcodes |
| **Buffer** | A pooled byte slice encoder that produces binary opcode frames |
| **Frame** | A complete binary message: `[opcode][fields...]` |
| **Mount** | The initial render: component function is called, tree is serialized, and opcodes are sent |
| **Flush** | Clearing a signal's dirty flag after its delta has been emitted |
| **Observer** | A callback registered on a signal via `Observe()` that fires on value change |
| **Node Registry** | The WASM client's `map[uint32]js.Value` mapping server node IDs to browser DOM elements |
| **WebTransport** | An HTTP/3-based bidirectional transport protocol built on QUIC |
| **QUIC** | A UDP-based transport protocol providing multiplexed streams with TLS 1.3 |
| **Passkey** | A WebAuthn credential stored on a hardware authenticator or platform |
| **Dead-Code Elimination** | The CSS generator only emits rules for classes actually referenced in code |
| **Zero Hydration** | CBS never re-executes server logic on the client — the WASM client only patches the DOM |
| **Backpressure** | The broadcast system's policy of dropping messages to slow subscribers |
| **`#cbs-root`** | The DOM element (`<div id="cbs-root">`) where the WASM client mounts the application |

---

*CBS is open source software. Module: `github.com/bytewiredev/bytewire`*
