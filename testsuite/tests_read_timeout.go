package testsuite

import (
	"time"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"

	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"

	"github.com/stretchr/testify/require"
)

func DoReadTimeoutTests(t *TestContext) {
	t.RequireCapability("read-timeout")

	t.Run("can time out", func(t *TestContext) {
		t.WithStreamEndpoint(func(e *stream.Endpoint) {
			opts := client.CreateStreamOpts{
				ReadTimeoutMS: ldvalue.NewOptionalInt(500),
			}
			t.WithTestClientStreamOpts(e, opts, func(r *client.ResponseStream) {
				_, err := e.AwaitConnection()
				require.NoError(t, err)

				e.SendChunk("data: Hello\n\n")
				time.Sleep(time.Second)

				t.RequireSpecificEvents(r,
					client.EventMessage{Data: "Hello"})

				_, err = r.AwaitMessage("error")
				require.NoError(t, err)

				_, err = e.AwaitConnection()
				require.NoError(t, err)
			})
		})
	})
}
