// Package protocol defines the CBS binary instruction set for DOM mutations.
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
)

// Client -> Server opcodes (0x10 - 0x1F)
const (
	// OpClientIntent relays a user interaction event to the server.
	// Format: [0x10][4B NodeID][1B EventType][payload bytes]
	OpClientIntent byte = 0x10

	// OpClientNav signals that the user navigated (popstate / link click).
	// Format: [0x11][UTF-8 path bytes]
	OpClientNav byte = 0x11
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
