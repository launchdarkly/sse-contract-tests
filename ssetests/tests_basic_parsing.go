package ssetests

import "fmt"

func DoBasicParsingTests(t *T) {
	t.Run("one-line message in one chunk", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "Hello"})
	})

	t.Run("one-line message in two chunks", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hel")
		t.SendOnStream("lo\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "Hello"})
	})

	t.Run("two one-line messages in one chunk", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hello\n\ndata: World\n\n")
		t.RequireSpecificEvents(
			EventMessage{Data: "Hello"},
			EventMessage{Data: "World"})
	})

	t.Run("one two-line message in one chunk", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hello\ndata:World\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "Hello\nWorld"})
	})

	t.Run("empty data", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data:\n\n")
		t.RequireSpecificEvents(EventMessage{Data: ""})
	})

	t.Run("event with specific type", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType("greeting")
		t.SendOnStream("event: greeting\ndata: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("default event type", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "message", Data: "Hello"})
	})

	t.Run("event with ID", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("id: abc\ndata: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{ID: "abc", Data: "Hello"})
	})

	t.Run("event with type and ID", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType("greeting")
		t.SendOnStream("event: greeting\nid: abc\ndata: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
	})

	t.Run("fields in reverse order", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType("greeting")
		t.SendOnStream("data: Hello\nid: abc\nevent: greeting\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
	})

	t.Run("unknown field is ignored", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType("greeting")
		t.SendOnStream("event: greeting\ncolor: blue\ndata: Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("fields without leading space", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType("greeting")
		t.SendOnStream("event:greeting\ndata:Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: "greeting", Data: "Hello"})
	})

	t.Run("fields with extra leading space", func(t *T) {
		t.StartSSEClient()
		t.TellClientToExpectEventType(" greeting")
		t.SendOnStream("event:  greeting\ndata:  Hello\n\n")
		t.RequireSpecificEvents(EventMessage{Type: " greeting", Data: " Hello"})
	})

	t.Run("multi-byte characters", func(t *T) {
		t.StartSSEClient()
		t.SendOnStream("data: €豆腐\n\n")
		t.RequireSpecificEvents(EventMessage{Data: "€豆腐"})
	})

	t.Run("many messages in rapid succession", func(t *T) {
		// This test verifies that the SSE client delivers messages in the same order they were received
		messageCount := 100
		allMessages := ""
		var expected []EventMessage
		for i := 0; i < messageCount; i++ {
			data := fmt.Sprintf("message %d", i)
			allMessages += fmt.Sprintf("data: %s\n\n", data)
			expected = append(expected, EventMessage{Data: data})
		}

		t.StartSSEClient()
		t.SendOnStream(allMessages)
		t.RequireSpecificEvents(expected...)
	})

	// The following test is based on one that's in the js-eventsource unit tests. While it works there,
	// it does not (cannot?) work in Ruby, and possibly some other platforms where there's no native
	// "non-string binary data" type. If that's true, we should probably delete this.
	// t.Run("multi-byte characters sent in single-byte pieces", func(t *T) {
	// 	   t.StartSSEClient()
	//     t.SendSplit("data: €豆腐\n\n", 1, time.Millisecond*20)
	//     t.RequireSpecificEvents(e, EventMessage{Data: "€豆腐"})
	// })
}
