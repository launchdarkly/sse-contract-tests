package testsuite

import (
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"
)

func DoBasicParsingTests(t *TestContext) {
	t.Run("one-line message in one chunk", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "Hello"})
		})
	})

	t.Run("one-line message in two chunks", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hel")
			e.SendChunk("lo\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "Hello"})
		})
	})

	t.Run("two one-line messages in one chunk", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\n\ndata: World\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "Hello"},
				client.EventMessage{Data: "World"})
		})
	})

	t.Run("one two-line message in one chunk", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\ndata:World\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "Hello\nWorld"})
		})
	})

	t.Run("empty data", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data:\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: ""})
		})
	})

	t.Run("event with specific type", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("event: greeting\ndata: Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: "greeting", Data: "Hello"})
		})
	})

	t.Run("event with ID", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("id: abc\ndata: Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{ID: "abc", Data: "Hello"})
		})
	})

	t.Run("event with type and ID", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("event: greeting\nid: abc\ndata: Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
		})
	})

	t.Run("fields in reverse order", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\nid: abc\nevent: greeting\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: "greeting", ID: "abc", Data: "Hello"})
		})
	})

	t.Run("unknown field is ignored", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("event: greeting\ncolor: blue\ndata: Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: "greeting", Data: "Hello"})
		})
	})

	t.Run("fields without leading space", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("event:greeting\ndata:Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: "greeting", Data: "Hello"})
		})
	})

	t.Run("fields with extra leading space", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("event:  greeting\ndata:  Hello\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Type: " greeting", Data: " Hello"})
		})
	})

	t.Run("multi-byte characters", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: €豆腐\n\n")

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "€豆腐"})
		})
	})

	t.Run("multi-byte characters sent in single-byte pieces", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendSplit("data: €豆腐\n\n", 1, time.Millisecond*20)

			t.RequireSpecificEvents(r,
				client.EventMessage{Data: "€豆腐"})
		})
	})
}
