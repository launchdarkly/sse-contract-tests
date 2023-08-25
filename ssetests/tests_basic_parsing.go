package ssetests

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/stretchr/testify/assert"
	"time"

	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"
)

func generateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func DoBasicParsingTests(t *ldtest.T) {
	t.Run("one-line message in one chunk", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("two messages spanning 3 chunks with shared chunk", func(t *ldtest.T) {
		// This test primarily tests situations where the implementation over allocates buffers to decrease the total
		// number of buffers. This test helps ensure that the used size in the buffer is properly tracked.
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data: test")
		stream.Send("test\n\ndata:")
		stream.Send("test" + "\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "testtest"})
		client.RequireSpecificEvents(t, EventMessage{Data: "test"})
	})

	t.Run("large message in one chunk", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		randomData := generateRandomString(5 * 1024 * 1024)
		stream.Send("data: " + randomData + "\n\n")
		actual := client.RequireEvent(t)
		// Does not use RequireSpecificEvents, because then it would print megabytes of text.
		if actual.Data != (randomData) {
			assert.Fail(t, "Random message data did not match.")
		}
	})

	t.Run("large message in two chunks", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		randomDataA := generateRandomString(5 * 1024 * 1024)
		randomDataB := generateRandomString(5 * 1024 * 1024)
		stream.Send("data: " + randomDataA)
		stream.Send(randomDataB + "\n\n")
		// Does not use RequireSpecificEvents, because then it would print megabytes of text.
		actual := client.RequireEvent(t)
		if actual.Data != (randomDataA + randomDataB) {
			assert.Fail(t, "Random message data did not match.")
		}
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

	t.Run("ID field is ignored if it contains a null", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("id: a\x00bc\ndata: Hello\n\n")
		client.RequireSpecificEvents(t, EventMessage{Data: "Hello"})
	})

	t.Run("last ID persists if not overridden by later event", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("id: abc\ndata: first\n\n")
		stream.Send("data: second\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{ID: "abc", Data: "first"},
			EventMessage{ID: "abc", Data: "second"},
		)
	})

	t.Run("last ID can be overridden by an empty value", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("id: abc\ndata: first\n\n")
		stream.Send("id: \ndata: second\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{ID: "abc", Data: "first"},
			EventMessage{Data: "second"},
		)
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

	t.Run("field with no colon", func(t *ldtest.T) {
		// A line that says only "data" should be equivalent to "data:". Here we'll send two
		// events as follows:
		//
		//     data
		//
		//     data
		//     data
		//
		// The first of those should translate into an event with empty data. The second is an
		// event with a single newline in the data, just as it would if each "data" was "data:".
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send("data\n\ndata\ndata\n\n")
		client.RequireSpecificEvents(t,
			EventMessage{Data: ""},
			EventMessage{Data: "\n"},
		)
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
