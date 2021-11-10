# Contributing to this project
 
## Submitting bug reports and feature requests

The LaunchDarkly SDK team monitors the [issue tracker](https://github.com/launchdarkly/sse-contract-tests/issues) in this repository. Bug reports and feature requests specific to this project should be filed in this issue tracker. The SDK team will respond to all newly filed issues within two business days.

## Submitting pull requests
 
We encourage pull requests and other contributions from the community. Before submitting pull requests, ensure that all temporary or unintended code is removed. Don't worry about adding reviewers to the pull request; the LaunchDarkly SDK team will add themselves. The SDK team will acknowledge all pull requests within two business days.
 
## Build instructions
 
### Prerequisites

This project should be built against Go 1.14 or newer.

### Building

To build the project:
```
make
```

To build the Docker image:
```
make docker-build
```

To run the linter:
```
make lint
```

### Testing

Currently the CI build for this project consists of a smoke test where the tool is built in Docker and then run against a fake service that deliberately returns an error-- proving that the code at least builds, executes, and makes the expected initial status request.

To run this test locally:
```
make docker-smoke-test
```

## Pushing to Docker

To update the published Docker image, first update `DOCKER_IMAGE_MAJOR_VERSION`, `DOCKER_IMAGE_MINOR_VERSION`, and `DOCKER_IMAGE_PATCH_VERSION` as appropriate in `Makefile`. Then run:
```
make docker-push
```

This will require you to log into the `ldcircleci` Docker account.

## Writing tests

The test suite is written in Go code, in the `ssetests` package.

It does not use the Go test runner, but the API is deliberately similar to Go's `testing` package. The `ssetests.T` type implements the same basic methods as `testing.T`, so you can use test assertion libraries like `github.com/stretchr/testify/assert`. It also provides methods for managing the mock stream that the test harness creates for each test, and the SSE client that the test harness manages through the test service.

Tests will generally start by calling `StartSSEClient` or `StartSSEClientOptions`. They can then control the mock stream with methods such as `SendOnStream` and `BreakStreamConnection`, and declare expectations about what the SSE client should receive with methods such as `RequireEvent`.

Any test of extended capabilities that are not required for every SSE implementation should start by calling `RequireCapability`, causing that test (or group of tests) to be skipped if the test service did not declare that capability.

