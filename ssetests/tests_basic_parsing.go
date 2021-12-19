package ssetests

import (
	"fmt"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
)

func DoBasicParsingTests(t *ldtest.T) {
	t.Run("one-line message in one chunk", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("one-line message in two chunks", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hel")
		stream.Send("lo\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("two one-line messages in one chunk", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\n\ndata: World\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{Data: "Hello"},
			EventMessage{Data: "World"})
	})

	t.Run("one two-line message in one chunk", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\ndata:World\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello\nWorld"})
	})

	t.Run("empty data", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data:\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: ""})
	})

	t.Run("event with specific type", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send("event: greeting\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("default event type", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "message", Data: "Hello"})
	})

	t.Run("event with ID", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("id: abc\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{ID: "abc", Data: "Hello"})
	})

	t.Run("event with type and ID", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send("event: greeting\nid: abc\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
	})

	t.Run("fields in reverse order", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send("data: Hello\nid: abc\nevent: greeting\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
	})

	t.Run("unknown field is ignored", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send("event: greeting\ncolor: blue\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("fields without leading space", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, "greeting")
		stream.Send("event:greeting\ndata:Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("fields with extra leading space", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		client.BePreparedToReceiveEventType(t, " greeting")
		stream.Send("event:  greeting\ndata:  Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Type: " greeting", Data: " Hello"})
	})

	t.Run("multi-byte characters", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: €豆腐\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "€豆腐"})
	})

	t.Run("many messages in rapid succession", func(t *ldtest.T) {
		// This test verifies that the SSE client delivers messages in the same order they were received
		messageCount := 100
		allMessages := ""
		var expected []EventMessage
		for i := 0; i < messageCount; i++ {
			data := fmt.Sprintf("message %d", i)
			allMessages += fmt.Sprintf("data: %s\n\n", data)
			expected = append(expected, EventMessage{Data: data})
		}

		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(allMessages)
		client.RequireSpecificEvents(t, expected...)
	})

	t.Run("multi-byte characters sent in single-byte pieces", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.SendInChunks("data: €豆腐\n\n", 1, time.Millisecond*20)
		client.RequireSpecificEvents(t, EventMessage{Data: "€豆腐"})
	})
}
