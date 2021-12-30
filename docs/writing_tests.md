# Writing tests

The test suite is written in Go code, in the `ssetests` package.

It does not use the Go test runner, but the API is deliberately similar to Go's `testing` package. The `ssetests.T` type implements the same basic methods as `testing.T`, so you can use test assertion libraries like `github.com/stretchr/testify/assert`. It also provides methods for managing the mock stream that the test harness creates for each test, and the SSE client that the test harness manages through the test service.

Tests will generally start by calling `StartSSEClient` or `StartSSEClientOptions`. They can then control the mock stream with methods such as `SendOnStream` and `BreakStreamConnection`, and declare expectations about what the SSE client should receive with methods such as `RequireEvent`.

Any test of extended capabilities that are not required for every SSE implementation should start by calling `RequireCapability`, causing that test (or group of tests) to be skipped if the test service did not declare that capability.
