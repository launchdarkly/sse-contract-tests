# Change log

All notable changes to the project will be documented in this file. This project adheres to [Semantic Versioning](http://semver.org).


## [1.1.0] - 2021-12-03
### Added:
- New test "multi-byte characters sent in single-byte pieces" in "basic parsing". This verifies that the client correctly parses multi-byte UTF-8 characters if they are split across chunks of the stream.
- New test "one-line event" in each of the "linefeeds" groups ("LF separator", etc.).

## [1.0.0] - 2021-11-29
Initial stable release.
