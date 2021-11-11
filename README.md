# SSE Contract Tests

Implementations of the [Server-Sent Events](https://en.wikipedia.org/wiki/Server-sent_events) protocol must be able to handle many variations of data ordering, network behavior, etc. Providing thorough test coverage of these can be tedious. This project provides a test harness mechanism for running a standardized test suite against any SSE implementation.

This tool was developed to facilitate cross-platform testing of the LaunchDarkly SDKs, all of which rely on SSE for core functionality. However, it can be used for any SSE implementation. At a minimum, it will verify behaviors that are part of the canonical SSE [specification](https://html.spec.whatwg.org/multipage/server-sent-events.html); it can optionally also verify extended capabilities that some SSE implementations provide.

To use this tool, you must first implement a small web service that exercises the features of your SSE implementation. The behavior of the service endpoints is described below. After starting the service, run the test harness and give it the base URL of the test service. The test harness will start its own HTTP server to provide SSE data, which it will then ask the test service to connect to and read from.

## Test harness command line

```shell
./sse-contract-tests --url <test service base URL> [other options]
```

Options besides `--url`:

* `--host <NAME>` - sets the hostname to use in callback URLs, if not the same as the host the test service is running on (default: localhost)
* `--port <PORT>` - sets the callback port that test services will connect to (default: 8111)
* `--run <REGEX>` - skips any tests whose names do not match the specified regex (can specify more than one)
* `--skip <REGEX>` - skips any tests whose names match the specified regex (can specify more than one)
* `--debug` - enables verbose logging of test actions for failed tests
* `--debug-all` - enables verbose logging of test actions for all tests

For `--run` and `--skip`, the name of a test is the string that appears between brackets in the test output. This may have multiple segments delimited by slashes, such as `name of test category/name of subtest`.

## Running with Docker

In a CI job for an SSE implementation, the most convenient way to run the test suite is with Docker. The executable test harness is published as the Docker image `ldcircleci/sse-contract-tests:1`. This major version number will be updated if there is ever a non-backward-compatible change.

Since the test harness and the test service need to be able to make requests to each other, you will either need to use host networking or create a Docker bridge network. The latter is the only way to do it in CircleCI (you will also need to use `setup_remote_docker` to enable this in CircleCI).

Because the steps for doing this are basically always the same except for the initial step of building the test service, the tool can provide a script for running itself. It works like this:

First, build the Docker image for your test service. It should be configured to run the service as soon as the container is started.

Then, run this command, where `<SERVICENAME>` is the name of the Docker image you just built, `<SERVICEPORT>` is the port it will listen on, and `<OTHER_TEST_PARAMS>` are any other command-line parameters you want to pass to the test tool:

```
docker run ldcircleci/sse-contract-tests:1 \
  --url http://<SERVICENAME>:<SERVICEPORT> <OTHER_TEST_PARAMS> \
  --output-docker-script 1
```

The output of this command is a script which, if piped back into the shell (`| bash`), will take care of setting up the shared network, starting the test service container, running the test harness container, and cleaning up afterward. It will also dump the log output of the test service container if any tests failed.

## Test service endpoints

### Status resource: `GET /`

This resource should return a 200 status to indicate that the service has started. Optionally, it can also return a JSON object in the response body, with the following properties:

* `capabilities`: An array of strings describing optional features that this SSE implementation supports:
  * `"comments"`: The SSE client allows the caller to read comment lines. All SSE implementations must be able to handle comment lines, but many of them will simply discard the comments and not allow them to be seen.
  * `"headers"`: The SSE client can be configured to send custom headers.
  * `"last-event-id"`: The SSE client can be configured to send a specific `Last-Event-Id` value in its initial HTTP request.
  * `"post"`: The SSE client can be configured to send a `POST` request with a body instead of a `GET`.
  * `"read-timeout"`: The SSE client can be configured with a specific read timeout (a.k.a. socket timeout).
  * `"report"`: The SSE client can be configured to send a `REPORT` request with a body instead of a `GET`.
  * `"restart"`: The caller can tell the SSE client at any time to disconnect and retry.

The test harness will use the `capabilities` information to decide whether to run optional parts of the test suite that relate to those capabilities.

### Create stream: `POST /`

A `POST` request indicates that the test harness wants to start an instance of the SSE client. The request body is a JSON object with the following properties. All of the properties except `streamUrl` and `callbackUrl` are optional.

* `streamUrl`: The URL of an SSE endpoint created by the test harness.
* `callbackUrl`: The URL of a callback endpoint created by the test harness (see "Callback endpoint" below).
* `tag`: A string describing the current test, if desired for logging.
* `initialDelayMs`: An optional integer specifying the initial reconnection delay parameter, in milliseconds. Not all SSE client implementations allow this to be configured, but the test harness will send a value anyway in an attempt to avoid having reconnection tests run unnecessarily slowly.
* `lastEventId`: An optional string which should be sent as the `Last-Event-Id` header in the initial HTTP request. The test harness will only set this property if the test service has the `"last-event-id"` capability.
* `headers`: A JSON object containing additional HTTP header names and string values. The SSE client should be configured to add these headers to its HTTP requests. The test harness will only set this property if the test service has the `"headers"` capability. Header names can be assumed to all be lowercase.
* `method`: A string specifying an HTTP method to use instead of `GET`. The test harness will only set this property if the test service has the `"post"` or `"report"` capability.
* `body`: A string specifying data to be sent in the HTTP request body. The test harness will only set this property if the test service has the `"post"` or `"report"` capability.

The response to a valid request is any HTTP `2xx` status, with a `Location` header whose value is the URL of the test service resource representing this instance (that is, the one that would be used for "Close stream" or "Send command" as described below).

If any parameters are invalid, return HTTP `400`.

### Send command: `POST <URL of stream instance>`

A `POST` request to the resource that was returned by "Create stream" means the test harness wants to do something to an existing SSE client instance. The request body is a JSON object with the following property:

* `command`: Currently the only supported value is `"restart"`, meaning the stream should be disconnected and reconnected with the same stream URL. This will only be sent if the test service has the `"restart"` capability.

Return any HTTP `2xx` status, `400` for an unrecognized command, or `404` if there is no such stream.

If the SSE implementation does not support any special commands, then the test service doesn't need to implement this endpoint.

### Close stream: `DELETE <URL of stream instance>`

A `DELETE` request to the resource that was returned by "Create stream" means the test harness is done with this SSE client instance and the test service should stop it.

Return any HTTP `2xx` status, or `404` if there is no such stream.

## Callback endpoint

When the test harness tells the test service to create a stream, it provides a callback URL that is specific to that stream. The test service should make `POST` requests to this URL to deliver information about the status of the stream. The request body is always a JSON object, which can be one of the following:

#### `hello` message

The test service must send this message first when it has been told to create a stream. This just tells the test harness that the stream is being created, regardless of whether it succeeds or fails.

```json
{ "kind": "hello" }
```

#### `event` message

This message indicates that the test service has received an event from the SSE stream. The `type`, `data`, and `id` fields correspond to the fields of an SSE event. All but `data` are optional.

```json
{
  "kind": "event",
  "event": {
    "type": "put",
    "data": "my-event-data",
    "id": "my-event-ID"
  }
}
```

#### `comment` message

This message indicates that the test service has received a comment from the SSE stream (if the SSE implementation allows the caller to see comments).

```json
{
  "kind": "comment",
  "comment": "the comment text"
}
```

#### `error` message

This message indicates that the test service has received an error from the SSE stream.

```json
{
  "kind": "error",
  "comment": "the error message"
}
```
