# Change log

All notable changes to the project will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).


## [2.31.0](https://github.com/launchdarkly/sse-contract-tests/compare/v2.30.0...v2.31.0) (2024-12-24)


### Features

* Add large message test and additional chunked message test. ([f067d20](https://github.com/launchdarkly/sse-contract-tests/commit/f067d208247e594e3c5af6e537f6a1a739995899))
* Add test to verify 204 halts re-connection ([#20](https://github.com/launchdarkly/sse-contract-tests/issues/20)) ([01953fa](https://github.com/launchdarkly/sse-contract-tests/commit/01953fa87a2c98fdb4ed8b84fbde4ab84e7d2f08))
* Test large message payloads. ([d9f9282](https://github.com/launchdarkly/sse-contract-tests/commit/d9f928264ee764373124926288f382653151f3c3))

## [2.3.0] - 2023-08-25
### Added:
- Added a tests which use large message sizes. (5-10MB).
- Added a test which chunks 2 messages over 3 chunks, with the middle chunk being shared.

## [2.2.0] - 2023-06-12
### Added:
- Add capability-protected test to ensure a 204 can direct the eventsource to stop retrying disconnects.
- Add test to ensure an empty location header is handled with an appropriate error.

## [2.1.1] - 2022-02-28
### Fixed:
- Fixed handling of spaces in command-line argument values

## [2.1.0] - 2022-02-16
### Added:
- The downloadable artifacts now include arm64 builds.

### Fixed:
- If the test service neglects to send a required `event` field in a callback, it now produces a clear error rather than a panic.

## [2.0.0] - 2022-01-05
This new major version release is due to a non-backward-compatible change in the test service protocol (regarding the `listen` command). Be aware that projects that currently pass the tests might fail in the future due to a new mandatory test in a minor version release, if the new test is for behavior that was already required by the specification. See `docs/deploying.md` for more about versioning of this tool.

### Added:
- New mandatory test: the client must automatically follow an HTTP 301 or 307 redirect.
- New mandatory test: the client must ignore any `id` field whose value contains a null.
- New mandatory test: the client must allow the last event ID to be overwritten with an empty value if an event explicitly provides an `id` field with an empty value.
- New mandatory test: any non-empty line that does not contain a colon must be treated as a field with an empty value (e.g., `data` should be treated the same as `data:`).
- New mandatory test: the client must not retain any part of a partly-parsed but incomplete message if the connection is dropped.

### Changed:
- For test service implementations that recognize the `listen` command, the schema for the request body has been changed.
- The test framework has been rewritten using a somewhat different approach similar to https://github.com/launchdarkly/sdk-test-harness, with a cleaner separation of concerns. This should make the test logic easier to follow.

## [1.1.0] - 2021-12-03
### Added:
- New test "multi-byte characters sent in single-byte pieces" in "basic parsing". This verifies that the client correctly parses multi-byte UTF-8 characters if they are split across chunks of the stream.
- New test "one-line event" in each of the "linefeeds" groups ("LF separator", etc.).

## [1.0.0] - 2021-11-29
Initial stable release.
