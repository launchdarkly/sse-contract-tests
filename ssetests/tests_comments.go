package ssetests

import (
	"github.com/launchdarkly/sse-contract-tests/framework/ldtest"

	"github.com/stretchr/testify/assert"
)

func DoCommentTests(t *ldtest.T) {
	t.RequireCapability("comments")

	t.Run("single comment", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)
		stream.Send(":Hello\n")
		c := client.RequireComment(t)
		assert.Equal(t, "Hello", c)
	})

	t.Run("two comments in a row", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)

		stream.Send(":Hello\n")
		c1 := client.RequireComment(t)
		assert.Equal(t, "Hello", c1)

		stream.Send(":World\n")
		c2 := client.RequireComment(t)
		assert.Equal(t, "World", c2)
	})

	t.Run("comment before event", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)

		stream.Send(":Hello\n")
		stream.Send("data: Hello\n\n")

		c := client.RequireComment(t)
		assert.Equal(t, "Hello", c)

		ev := client.RequireEvent(t)
		assert.Equal(t, "message", ev.Type)
		assert.Equal(t, "Hello", ev.Data)
	})

	t.Run("comment after event", func(t *ldtest.T) {
		_, stream, client := NewStreamAndSSEClient(t)

		stream.Send("data: Hello\n\n")
		stream.Send(":Hello\n")

		ev := client.RequireEvent(t)
		assert.Equal(t, "message", ev.Type)
		assert.Equal(t, "Hello", ev.Data)

		c := client.RequireComment(t)
		assert.Equal(t, "Hello", c)
	})
}
