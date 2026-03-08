# Bytewire Architecture

## Overview

Bytewire is a server-driven UI framework. The server owns all state and logic; the browser runs a thin WASM client that patches the DOM. Communication uses a compact binary protocol over WebTransport (HTTP/3 + QUIC) with WebSocket fallback.

## System Architecture

```mermaid
graph TB
    subgraph Browser
        UI[Browser DOM]
        WASM[WASM Client]
        ED[Event Delegation]
        WT_C[WebTransport / WebSocket]
    end

    subgraph Server ["Server (Go)"]
        ENG[Engine]
        SESS[Session Manager]
        COMP[Component Tree]
        SIG[Reactive Signals]
        VDOM[Virtual DOM]
        DIFF[Diff / Flush]
        PROTO[Protocol Encoder]
        WT_S[WebTransport / WebSocket]
    end

    UI -->|user interaction| ED
    ED -->|binary intent| WT_C
    WT_C -->|OpClientIntent / OpClientNav| WT_S
    WT_S --> ENG
    ENG --> SESS
    SESS --> COMP
    COMP --> SIG
    SIG -->|dirty tracking| VDOM
    VDOM --> DIFF
    DIFF --> PROTO
    PROTO -->|binary opcodes| WT_S
    WT_S -->|binary opcodes| WT_C
    WT_C --> WASM
    WASM -->|DOM patches| UI
```

## Request/Response Lifecycle

```mermaid
sequenceDiagram
    participant B as Browser
    participant W as WASM Client
    participant T as Transport (WT/WS)
    participant E as Engine
    participant S as Session
    participant C as Component

    B->>W: User clicks button
    W->>W: findBWNode() -> nodeID
    W->>T: OpClientIntent [nodeID, EventClick]
    T->>E: Binary frame
    E->>S: HandleIntent()
    S->>C: Invoke OnClick handler
    C->>C: signal.Set(newValue)
    C->>C: Dirty signals trigger observers
    Note over S: Flush loop (batched)
    S->>S: Diff virtual DOM
    S->>T: Binary opcodes (OpSetAttr, OpUpdateText, ...)
    T->>W: Binary frame
    W->>B: Apply DOM patches
```

## Package Structure

```mermaid
graph LR
    subgraph Public API
        engine[engine]
        dom[dom]
        style[style]
        components[components]
        router[router]
    end

    subgraph Internal
        protocol[protocol]
        wasm[wasm]
        ratelimit[ratelimit]
        metrics[metrics]
        plugin[plugin]
        devcert[devcert]
        webauthn[webauthn]
    end

    engine --> dom
    engine --> protocol
    engine --> ratelimit
    engine --> metrics
    engine --> plugin
    engine --> devcert
    engine --> webauthn
    dom --> protocol
    components --> dom
    components --> style
    router --> engine
    wasm --> protocol
```

| Package | Purpose |
|---------|---------|
| `engine` | WebTransport/WebSocket server, session lifecycle, flush loop |
| `dom` | Virtual DOM nodes, signals, reactive primitives (`If`, `For`, `TextF`) |
| `protocol` | Binary opcode encoding/decoding, buffer pool |
| `wasm` | Browser WASM client: DOM patching, event delegation, transport |
| `style` | Type-safe Tailwind-style CSS utility classes |
| `components` | Reusable UI: forms, table, modal, badge, alert, spinner, charts |
| `router` | Server-side URL routing with `:param` extraction |
| `ratelimit` | Per-session token bucket rate limiter |
| `metrics` | Prometheus-compatible counter, gauge, histogram, registry |
| `plugin` | Lifecycle hooks (connect, mount, disconnect) |
| `devcert` | Auto-generated ephemeral TLS certificates for development |
| `webauthn` | Passkey/WebAuthn credential store and challenge flow |

## Transport Layer

```mermaid
graph TD
    subgraph Primary ["WebTransport (QUIC/UDP)"]
        WT[HTTP/3 Server :4433]
        WT -->|Upgrade /bw| SESS1[Session]
        SESS1 -->|Bidirectional streams| IO1[Read intents / Write opcodes]
    end

    subgraph Fallback ["WebSocket (TCP)"]
        WS[HTTP Server :8080]
        WS -->|Upgrade /bw-ws| SESS2[Session]
        SESS2 -->|Binary frames| IO2[Read intents / Write opcodes]
    end

    CLIENT[Browser] -->|WebTransport supported?| Primary
    CLIENT -->|No WebTransport| Fallback
```

- **WebTransport**: QUIC-based, multiplexed, no head-of-line blocking. Primary transport.
- **WebSocket**: TCP-based fallback. Server sets `TCP_NODELAY` to minimize Nagle buffering.
- Both transports use identical length-prefixed binary framing: `[4B length][payload]`.

## Binary Protocol

All DOM mutations are encoded as compact binary opcodes. Each frame is length-prefixed.

```mermaid
graph LR
    subgraph Frame
        LEN[4B Length]
        OP[1B Opcode]
        BODY[Variable payload]
    end
    LEN --> OP --> BODY
```

### Server -> Client Opcodes

| Opcode | Hex | Payload | Description |
|--------|-----|---------|-------------|
| UpdateText | `0x01` | nodeID + text | Set text content |
| SetAttr | `0x02` | nodeID + key + value | Set attribute (uses fast path for `id`, `class`) |
| RemoveAttr | `0x03` | nodeID + key | Remove attribute |
| InsertNode | `0x04` | nodeID + parentID + tag + attrs | Create element |
| RemoveNode | `0x05` | nodeID | Remove element from tree |
| ReplaceText | `0x06` | nodeID + offset + text | Surgical text splice |
| SetStyle | `0x07` | nodeID + prop + value | Set inline CSS property |
| PushHistory | `0x08` | URL path | Update browser URL via History API |
| Batch | `0x09` | nested opcodes | Atomic group applied together |
| Error | `0x0A` | message | Display error overlay |
| DevToolsState | `0x0B` | JSON snapshot | Session state for devtools |
| InsertText | `0x0E` | nodeID + parentID + text | Create text node with content (combined) |
| InsertHTML | `0x0F` | parentID + HTML string | Bulk HTML insert via template element |
| ClearChildren | `0x14` | parentID | Remove all children, bulk ID cleanup |
| SwapNodes | `0x15` | nodeA + nodeB | Swap two DOM nodes (optimized 2-element swap) |
| BatchText | `0x16` | count + [nodeID + text]... | Batched text updates in single frame |

### Client -> Server Opcodes

| Opcode | Hex | Payload | Description |
|--------|-----|---------|-------------|
| ClientIntent | `0x10` | nodeID + eventType + data | User event (click, input, submit) |
| ClientNav | `0x11` | URL path | Browser navigation (popstate, link click) |
| ClientAuth | `0x13` | credential data | WebAuthn response |

## Reactive System

```mermaid
graph TD
    SIG["Signal[T]"] -->|Set/Update| DIRTY[Mark dirty]
    DIRTY --> OBS[Notify observers]
    OBS --> TEXTF["TextF() - update text node"]
    OBS --> CLASSF["ClassF() - update class attr"]
    OBS --> IFC["If() - toggle subtree"]
    OBS --> FORC["For() - diff keyed list"]
    TEXTF --> BUF[Encode to protocol.Buffer]
    CLASSF --> BUF
    IFC --> BUF
    FORC --> BUF
    BUF --> FLUSH[Session flush loop]
    FLUSH -->|Binary frame| TRANSPORT[Transport]
```

### Signal Types

- **`Signal[T]`** -- Single value. `Set()`, `Update()`, `Get()`. Observers fire on change.
- **`ListSignal[T]`** -- Ordered collection. `Append()`, `Remove()`, `Clear()`. Used with `For()`.
- **`Computed[T]`** -- Derived value from other signals. Read-only.

### List Diffing (`For()`)

The `For()` primitive uses keyed reconciliation with LIS (Longest Increasing Subsequence) optimization:

```mermaid
graph TD
    NEW[New items list] --> KEYS[Extract keys]
    OLD[Old items map] --> KEYS
    KEYS --> ADD[New keys: InsertNode/InsertHTML]
    KEYS --> REM[Removed keys: RemoveNode]
    KEYS --> REORDER{Order changed?}
    REORDER -->|Yes| LIS[Compute LIS]
    LIS --> MOVE[Move only non-LIS items]
    REORDER -->|No| DONE[No moves needed]

    ADD --> BULK{20+ new items?}
    BULK -->|Yes| HTML[OpInsertHTML bulk path]
    BULK -->|No| NODES[Individual OpInsertNode]

    REM --> CLEAR{All removed?}
    CLEAR -->|Yes| CLEAROP[OpClearChildren]
    CLEAR -->|No| INDIVIDUAL[Individual OpRemoveNode]
```

## WASM Client

The browser-side WASM client is ~50KB and handles:

1. **Transport connection** -- WebTransport with WebSocket fallback, auto-reconnect with exponential backoff
2. **DOM patching** -- Applies binary opcodes directly to the DOM via `syscall/js`
3. **Event delegation** -- Single listener on `#bw-root` for click, input, submit
4. **Node registry** -- `map[uint32]js.Value` maps server-assigned IDs to DOM elements
5. **SPA routing** -- Intercepts `<a>` clicks with local hrefs, sends `OpClientNav`

### Performance Optimizations

- **`__bwId` JS property** on elements instead of `data-bw-id` attribute (avoids `getAttribute` overhead)
- **`setAttrFast`** uses direct property assignment for `id`, `class`, `className` instead of `setAttribute`
- **`__bwProcessHTML`** JS helper for bulk insert: parses HTML via `<template>`, collects IDs in a single JS loop
- **`__bwCollectIds`** JS helper for bulk removal: collects all descendant IDs as packed `Uint8Array`
- **`data-bw-tid`** proxy: parent element holds text node reference, avoiding wrapper elements

## Server Engine Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Init: NewServer()
    Init --> Listening: ListenAndServe()

    state Listening {
        [*] --> WaitConn
        WaitConn --> Upgrade: Client connects
        Upgrade --> SessionCreated: Create Session
        SessionCreated --> Hello: Send protocol handshake
        Hello --> Mount: Mount component tree
        Mount --> Active: Flush initial DOM

        state Active {
            [*] --> ReadIntent
            ReadIntent --> HandleIntent: Binary frame received
            HandleIntent --> SignalUpdate: Invoke handler
            SignalUpdate --> DirtyCheck: Mark signals dirty
            DirtyCheck --> Flush: Encode opcodes
            Flush --> SendOpcodes: Write to transport
            SendOpcodes --> ReadIntent
        }

        Active --> Disconnected: Connection lost
        Disconnected --> [*]: Cleanup session
    }
```

## SSR (Server-Side Rendering)

When enabled with `WithSSR()`, the initial HTTP response includes pre-rendered HTML:

```mermaid
sequenceDiagram
    participant B as Browser
    participant HTTP as HTTP Server
    participant WT as WebTransport

    B->>HTTP: GET /page
    HTTP->>HTTP: Mount component (noop writer)
    HTTP->>HTTP: dom.RenderHTML(root)
    HTTP->>B: HTML shell + pre-rendered content
    B->>B: Display immediately (no blank flash)
    B->>B: Load WASM
    B->>WT: Connect WebTransport
    WT->>B: Full interactive opcodes
    Note over B: hydrateExistingDOM() maps<br/>existing elements to node registry
```

## Reconnection

```mermaid
stateDiagram-v2
    [*] --> Connected
    Connected --> Lost: Connection drops
    Lost --> ShowOverlay: Display "Reconnecting..."
    ShowOverlay --> Retry: Wait (exponential backoff)
    Retry --> Connected: Success
    Retry --> Retry: Failed (1s, 2s, 4s, 8s, 10s max)
    Retry --> Fatal: 30s timeout
    Fatal --> [*]: "Please reload"

    state Connected {
        [*] --> Online
        Online --> DrainQueue: Flush offline queue
        DrainQueue --> Online
    }
```

The WASM client queues events during disconnection (persisted to `sessionStorage`) and drains them on reconnect.

## Chart Rendering (Hybrid Canvas)

Charts use a hybrid server-client architecture that rides on existing opcodes:

```mermaid
graph TD
    subgraph Server
        BS["BoxSignal[ChartData]"] -->|Observe| AJ[AttrJSON serializer]
        AJ -->|JSON string| SA[OpSetAttr: data-bw-chart-data]
    end

    subgraph Client ["WASM Client"]
        SA -->|setAttrFast| RC[renderChart dispatcher]
        RC -->|"bar"| BAR[renderBarChart]
        RC -->|"line"| LINE[renderLineChart]
        RC -->|"pie"| PIE[renderPieChart]
        RC -->|"sparkline"| SPARK[renderSparkLine]
        BAR --> CTX[Canvas 2D API]
        LINE --> CTX
        PIE --> CTX
        SPARK --> CTX
    end
```

**No new opcodes needed.** The server creates `<canvas>` elements with:
- `data-bw-chart` — chart type (bar, line, pie, sparkline)
- `data-bw-chart-data` — JSON-serialized data from a `BoxSignal`

When the signal changes, `AttrJSON` marshals the new data and sets it as a dirty attribute. The engine emits a standard `OpSetAttr`. The WASM client's `setAttrFast` intercepts `data-bw-chart-data` changes and calls `renderChart()`, which dispatches to the appropriate Canvas 2D renderer.

### Signal Types (Updated)

- **`Signal[T comparable]`** — Single comparable value. Deduplicates on `==`.
- **`BoxSignal[T any]`** — Single non-comparable value (e.g., structs with slices). Always notifies on Set.
- **`ListSignal[T any]`** — Ordered collection for use with `For()`.
- **`Computed[T]`** — Derived read-only value.

### Prefetch Warmup

On hover over SPA links (`<a data-bw-link>`), the WASM client sends an `OpClientIntent` with `EventMouseEnter`. The server's session handler detects this and calls `PrefetchRoute()`, which warms the Go runtime for that route path. This reduces navigation latency by pre-allocating memory and warming caches ahead of the actual click.
