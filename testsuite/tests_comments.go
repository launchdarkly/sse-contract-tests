package testsuite

import (
	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"

	"github.com/stretchr/testify/assert"
)

func DoCommentTests(t *TestContext) {
	t.RequireCapability("comments")

	t.Run("single comment", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk(":Hello\n")

			c := t.RequireComment(r)
			assert.Equal(t, "Hello", c)
		})
	})

	t.Run("two comments in a row", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk(":Hello\n")

			c1 := t.RequireComment(r)
			assert.Equal(t, "Hello", c1)

			e.SendChunk(":World\n")

			c2 := t.RequireComment(r)
			assert.Equal(t, "World", c2)
		})
	})

	t.Run("comment before event", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk(":Hello\n")
			e.SendChunk("data: Hello\n\n")

			c := t.RequireComment(r)
			assert.Equal(t, "Hello", c)

			ev := t.RequireEvent(r)
			assert.Equal(t, "message", ev.Type)
			assert.Equal(t, "Hello", ev.Data)
		})
	})

	t.Run("comment after event", func(t *TestContext) {
		t.WithEndpointAndClientStream(func(e *stream.Endpoint, r *client.ResponseStream) {
			e.SendChunk("data: Hello\n\n")
			e.SendChunk(":Hello\n")

			ev := t.RequireEvent(r)
			assert.Equal(t, "message", ev.Type)
			assert.Equal(t, "Hello", ev.Data)

			c := t.RequireComment(r)
			assert.Equal(t, "Hello", c)
		})
	})
}
