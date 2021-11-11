// Package framework contains the low-level implementation of test harness infrastructure
// that can be reused for different kinds of tests.
//
// The general model is:
//
// 1. The test harness communicates with a test service, which exposes a root endpoint
// for querying its status (GET) or creating some kind of entity within the test service
// (POST).
//
// 2. The test harness can expose any number of mock endpoints to receive requests from
// the test service.
//
// 3. There is a general notion of a test context which is similar to Go's *testing.T,
// allowing pieces of test logic to be associated with a test identifier and to accumulate
// success/failure results.
//
// The domain-specific code that knows what is being tested is responsible for providing
// the parameters to send to the test service, the HTTP handlers for handling requests to
// mock endpoints, and a domain-specific test API on top of the test context.
package framework
