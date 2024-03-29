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

To build the Docker image locally (note that we normally use a different mechanism for publishing releases in Docker):
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

## Publishing releases

We normally use our internal Releaser tool. This takes care of updating the changelog, the version string in `version.go`, and the Git release history, as well as publishing the Docker image; see scripts in `.ldrelease`.

If you need to do a release manually for whatever reason, the steps are:

1. Update `VERSION` and `CHANGELOG.md`. Push these changes and create a version tag such as `v1.0.0`.
2. Use `docker login` to provide the credentials of the `ldcircleci` Docker account.
3. Run `make publish-release`.
4. Look in `./dist` for all `.tar.gz` and `.zip` files. These are the archives of executable binaries for various platforms. Attach these files to the GitHub release.

To do a dry run locally that builds all of the executables and the Docker image without publishing them, run `make build-release`. You can also use Releaser's dry-run mode to do the same for any branch in GitHub.
