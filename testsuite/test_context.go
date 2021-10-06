package testsuite

import (
	"errors"
	"fmt"
	"log"

	"github.com/launchdarkly/sse-contract-tests/client"
	"github.com/launchdarkly/sse-contract-tests/stream"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContext is used similarly to *testing.T in the test suite. It implements require.TestingT so
// we can use standard assertions from assert/require, has a Run method for subtests, and can skip
// tests that require capabilities the test service doesn't support.
type TestContext struct {
	client      *client.SSETestClient
	server      *stream.Server
	result      *Result
	testLogger  TestLogger
	debugLogger *log.Logger
	id          TestID
	failed      bool
	skipped     bool
	errors      []error
}

func Run(
	client *client.SSETestClient,
	server *stream.Server,
	result *Result,
	testLogger TestLogger,
	debugLogger *log.Logger,
	action func(*TestContext),
) {
	t := &TestContext{
		client:      client,
		server:      server,
		result:      result,
		testLogger:  testLogger,
		debugLogger: debugLogger,
	}
	t.run(action)
}

func (t *TestContext) HasCapability(capability string) bool {
	return t.client.HasCapability(capability)
}

func (t *TestContext) Errorf(format string, args ...interface{}) {
	t.failed = true
	err := fmt.Errorf(format, args...)
	t.errors = append(t.errors, err)
	t.testLogger.TestError(t.id, err)
}

func (t *TestContext) FailNow() {
	panic(t)
}

func (t *TestContext) Skip() {
	t.skipped = true
	panic(t)
}

func (t *TestContext) RequireCapability(capability string) {
	if !t.HasCapability(capability) {
		t.Skip()
	}
}

func (t *TestContext) Run(name string, action func(*TestContext)) {
	id := TestID{Path: append(t.id.Path, name)}
	t.testLogger.TestStarted(id)
	t1 := *t
	t1.id = id
	t1.run(action)
	if t1.skipped {
		t.testLogger.TestSkipped(id)
	} else {
		t.testLogger.TestFinished(id, t1.failed)
	}
}

func (t *TestContext) run(action func(*TestContext)) {
	defer func() {
		if r := recover(); r != nil {
			if t.skipped {

			} else {
				t.failed = true
				var addError error
				if _, ok := r.(*TestContext); ok {
					if len(t.errors) == 0 {
						addError = errors.New("test failed with no failure message")
					}
				} else {
					addError = fmt.Errorf("unexpected panic in test: %+v", r)
				}
				if addError != nil {
					t.errors = append(t.errors, addError)
					t.testLogger.TestError(t.id, addError)
				}
			}
		}
		result := TestResult{TestID: t.id, Errors: t.errors}
		t.result.Tests = append(t.result.Tests, result)
		if t.failed {
			t.result.Failures = append(t.result.Failures, result)
		}
	}()
	action(t)
}

func (t *TestContext) WithStreamEndpoint(action func(*stream.Endpoint)) {
	prefix := t.id.String() + " << "
	subLogger := log.New(t.debugLogger.Writer(), prefix, t.debugLogger.Flags())

	e := t.server.NewEndpoint(subLogger)
	defer e.Close()
	action(e)
}

func (t *TestContext) WithTestClientStream(e *stream.Endpoint, action func(*client.ResponseStream)) {
	opts := client.CreateStreamOpts{}
	t.WithTestClientStreamOpts(e, opts, action)
}

func (t *TestContext) WithTestClientStreamOpts(e *stream.Endpoint, opts client.CreateStreamOpts, action func(*client.ResponseStream)) {
	opts.URL = e.URL
	opts.Tag = t.id.String()
	r, err := t.client.CreateStream(opts)
	require.NoError(t, err)
	defer r.Close()

	_, err = r.AwaitMessage("hello")
	require.NoError(t, err)

	action(r)
}

func (t *TestContext) WithEndpointAndClientStream(action func(*stream.Endpoint, *client.ResponseStream)) {
	t.WithStreamEndpoint(func(e *stream.Endpoint) {
		t.WithTestClientStream(e, func(r *client.ResponseStream) {
			_, err := e.AwaitConnection()
			require.NoError(t, err)
			action(e, r)
		})
	})
}

func (t *TestContext) RequireEvent(r *client.ResponseStream) client.EventMessage {
	event, err := r.AwaitEvent()
	require.NoError(t, err)
	return event
}

func (t *TestContext) RequireSpecificEvents(r *client.ResponseStream, events ...client.EventMessage) {
	for _, expected := range events {
		if expected.Type == "" {
			expected.Type = "message"
		}
		actual := t.RequireEvent(r)
		if actual.Type == "" {
			actual.Type = "message"
		}
		assert.Equal(t, expected, actual)
	}
}

func (t *TestContext) RequireComment(r *client.ResponseStream) string {
	c, err := r.AwaitComment()
	require.NoError(t, err)
	return c
}
