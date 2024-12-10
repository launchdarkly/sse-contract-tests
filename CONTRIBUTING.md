# Contributing to this project
 
This page is for people doing development of the SDK test harness itself. See also the [general documentation](./docs/index.md) for how to use this tool, how to write test services for it, and how the individual tests in the test harness are written.

## Submitting bug reports and feature requests

The LaunchDarkly SDK team monitors the [issue tracker](https://github.com/launchdarkly/sse-contract-tests/issues) in this repository. Bug reports and feature requests specific to this project should be filed in this issue tracker. The SDK team will respond to all newly filed issues within two business days.

## Submitting pull requests
 
We encourage pull requests and other contributions from the community. Before submitting pull requests, ensure that all temporary or unintended code is removed. Don't worry about adding reviewers to the pull request; the LaunchDarkly SDK team will add themselves. The SDK team will acknowledge all pull requests within two business days.
 
## Build instructions
 
### Prerequisites

This project should be built against Go 1.17 or newer.

### Building

To build the project:
```
make
```

To run the linter:
```
make lint
```
