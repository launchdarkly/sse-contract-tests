package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"

	"github.com/stretchr/testify/assert"
)

// UTF-8 BOM (Byte Order Mark) is the byte sequence 0xEF 0xBB 0xBF (U+FEFF)
const utf8BOM = "\xEF\xBB\xBF"

// DoBOMTests verifies that SSE implementations correctly handle the UTF-8 BOM according to the spec:
// https://html.spec.whatwg.org/multipage/server-sent-events.html
//
// The spec states: "Streams must be decoded using the UTF-8 decode algorithm.
// If the stream contains a BOM (U+FEFF), it must be stripped."
func DoBOMTests(t *ldtest.T) {
	t.RequireCapability("bom")

	t.Run("BOM at start of stream is stripped", func(t *ldtest.T) {
		// Per spec, a BOM at the very beginning of the stream should be ignored
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(utf8BOM + "data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("BOM stripped before first event with type and ID", func(t *ldtest.T) {
		// Verify BOM stripping works with full event fields
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send(utf8BOM + "event: greeting\nid: abc\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
	})

	t.Run("BOM stripped with multiple messages", func(t *ldtest.T) {
		// BOM should only be stripped once at the beginning, not affect subsequent messages
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(utf8BOM + "data: First\n\ndata: Second\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{Data: "First"},
			EventMessage{Data: "Second"})
	})

	t.Run("BOM only stripped at stream start, not from later data", func(t *ldtest.T) {
		// A BOM appearing in the middle of the stream (not at the start) should be
		// treated as regular data
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: First\n\n")
		stream.Send("data: " + utf8BOM + "Second\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{Data: "First"},
			EventMessage{Data: utf8BOM + "Second"})
	})

	t.Run("BOM split across first two chunks", func(t *ldtest.T) {
		// Test that BOM stripping works even when the BOM bytes arrive in separate chunks.
		// Split: 0xEF 0xBB | 0xBF
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("\xEF\xBB")
		stream.Send("\xBF" + "data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("BOM split byte by byte across chunks", func(t *ldtest.T) {
		// Test BOM split into individual bytes across chunks
		// Split: 0xEF | 0xBB | 0xBF
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("\xEF")
		stream.Send("\xBB")
		stream.Send("\xBF")
		stream.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("only first BOM at start is stripped", func(t *ldtest.T) {
		// Per spec, "a BOM" (singular) should be stripped from the beginning.
		// A second BOM immediately in the data should be preserved.
		_, stream, client := NewStreamAndSSEClient(t)
		// First BOM is stripped, second BOM is part of the data field value
		stream.Send(utf8BOM + "data: " + utf8BOM + "Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: utf8BOM + "Hello"})
	})

	t.Run("no BOM - baseline test", func(t *ldtest.T) {
		// Baseline: verify normal operation without BOM
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("BOM with multi-line data", func(t *ldtest.T) {
		// Test BOM stripping with multi-line event data
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(utf8BOM + "data: Line1\ndata: Line2\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Line1\nLine2"})
	})

	t.Run("BOM with comment at start", func(t *ldtest.T) {
		// Test BOM followed by a comment. Behavior depends on whether the
		// implementation supports the "comments" capability.
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(utf8BOM + ":comment\ndata: Hello\n\n")

		if t.Capabilities().Has("comments") {
			// If comments are supported, expect the comment to be reported
			comment := client.RequireComment(t)
			assert.Equal(t, "comment", comment)
		}
		// Either way, we should receive the data event
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("BOM does not affect empty data field", func(t *ldtest.T) {
		// Test BOM with empty data
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(utf8BOM + "data:\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: ""})
	})
}
