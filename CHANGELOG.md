# Change log

All notable changes to the project will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).


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
