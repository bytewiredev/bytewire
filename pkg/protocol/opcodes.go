// Package protocol defines the Bytewire binary instruction set for DOM mutations.
//
// Every message is a compact byte sequence: [1B Opcode][4B NodeID][Payload].
// This eliminates JSON parsing overhead and enables zero-copy DOM patching.
package protocol

// Server -> Client opcodes (0x01 - 0x0F)
const (
	// OpUpdateText sets the textContent of a DOM node.
	// Format: [0x01][4B NodeID][UTF-8 text bytes]
	OpUpdateText byte = 0x01

	// OpSetAttr sets an HTML attribute on a DOM node.
	// Format: [0x02][4B NodeID][key bytes][0x00 separator][value bytes]
	OpSetAttr byte = 0x02

	// OpRemoveAttr removes an HTML attribute from a DOM node.
	// Format: [0x03][4B NodeID][key bytes]
	OpRemoveAttr byte = 0x03

	// OpInsertNode inserts a new element into the DOM tree.
	// Format: [0x04][4B NodeID][4B ParentID][4B SiblingID][1B TagLen][tag bytes][2B AttrCount][attrs...]
	// SiblingID of 0 means append as last child.
	OpInsertNode byte = 0x04

	// OpRemoveNode removes a DOM node and all its children.
	// Format: [0x05][4B NodeID]
	OpRemoveNode byte = 0x05

	// OpReplaceText is a targeted text replacement within a text node.
	// Format: [0x06][4B NodeID][4B Offset][4B Length][UTF-8 replacement bytes]
	OpReplaceText byte = 0x06

	// OpSetStyle sets a CSS property on a node's inline style.
	// Format: [0x07][4B NodeID][property bytes][0x00][value bytes]
	OpSetStyle byte = 0x07

	// OpPushHistory triggers a browser pushState for client-side routing.
	// Format: [0x08][UTF-8 path bytes]
	OpPushHistory byte = 0x08

	// OpBatch wraps multiple opcodes into a single atomic frame.
	// Format: [0x09][4B count][...nested opcodes]
	OpBatch byte = 0x09

	// OpError sends a server error message to the client for display.
	// Format: [0x0A][2B message length][UTF-8 message bytes]
	OpError byte = 0x0A

	// OpDevToolsState sends a JSON state snapshot to the client for DevTools inspection.
	// Format: [0x0B][4B JSON length][JSON bytes]
	OpDevToolsState byte = 0x0B

	// OpAuthChallenge sends an authentication challenge to the client.
	// Format: [0x0C][1B rpID length][rpID bytes][32B challenge]
	OpAuthChallenge byte = 0x0C

	// OpAuthResult sends the authentication result to the client.
	// Format: [0x0D][1B success (0/1)][token bytes]
	OpAuthResult byte = 0x0D

	// OpInsertText creates a text node and sets its content in one operation.
	// Combines OpInsertNode(#text) + OpUpdateText for better throughput.
	// Format: [0x0E][4B NodeID][4B ParentID][UTF-8 text bytes]
	OpInsertText byte = 0x0E

	// OpInsertHTML inserts a subtree via pre-rendered HTML.
	// The client uses innerHTML for O(1) DOM creation, then walks
	// data-bw-id attributes to populate the node registry.
	// Format: [0x0F][4B ParentID][UTF-8 HTML bytes]
	OpInsertHTML byte = 0x0F

	// OpClearChildren removes all children of a node in one operation.
	// Much faster than individual OpRemoveNode for bulk removal (e.g., clearing a list).
	// Format: [0x14][4B ParentID]
	OpClearChildren byte = 0x14

	// OpSwapNodes swaps two DOM nodes in one operation.
	// More efficient than two OpInsertNode moves for simple 2-element swaps.
	// Format: [0x15][4B nodeA][4B nodeB]
	OpSwapNodes byte = 0x15

	// OpBatchText updates multiple text nodes in a single frame.
	// Eliminates per-frame overhead when many signal-bound text nodes change at once.
	// Format: [0x16][2B count][4B nodeID | 2B textLen | text bytes]...
	OpBatchText byte = 0x16
)

// Client -> Server opcodes (0x10 - 0x1F)
const (
	// OpClientIntent relays a user interaction event to the server.
	// Format: [0x10][4B NodeID][1B EventType][payload bytes]
	OpClientIntent byte = 0x10

	// OpClientNav signals that the user navigated (popstate / link click).
	// Format: [0x11][UTF-8 path bytes]
	OpClientNav byte = 0x11

	// OpClientAuth sends authentication assertion data from the client.
	// Format: [0x13][2B credID len][credID][2B authData len][authData][2B clientDataJSON len][clientDataJSON][2B sig len][sig]
	OpClientAuth byte = 0x13
)

// EventType constants for OpClientIntent payloads.
const (
	EventClick      byte = 0x01
	EventInput      byte = 0x02
	EventSubmit     byte = 0x03
	EventFocus      byte = 0x04
	EventBlur       byte = 0x05
	EventKeyDown    byte = 0x06
	EventKeyUp      byte = 0x07
	EventMouseEnter byte = 0x08
	EventMouseLeave byte = 0x09
)
