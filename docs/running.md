# Running the tests

## Test harness command line

```shell
./sse-test-harness --url <test service base URL> [other options]
```

Options besides `--url`:

* `--host <NAME>` - sets the hostname to use in callback URLs, if not the same as the host the test service is running on (default: localhost)
* `--port <PORT>` - sets the callback port that test services will connect to (default: 8111)
* `--run <PATTERN>` - skips any tests whose names do not match the specified pattern (can specify more than one)
* `--skip <PATTERN>` - skips any tests whose names match the specified pattern (can specify more than one)
* `--stop-service-at-end` - tells the test service to exit after the test run
* `--debug` - enables verbose logging of test actions for failed tests
* `--debug-all` - enables verbose logging of test actions for all tests

For `--run` and `--skip`, the rules for pattern matching are as follows:

* The match is done againt the full path of the test. The full path is the string that appears between brackets in the test output. It may include slash-delimited subtests, such as `parent test name/subtest name/sub-subtest name`.
* If `--run` specifies a test that has subtests, then all of its subtests are also run.
* If `--skip` specifies a test that has subtests, then all of its subtests are also skipped.
