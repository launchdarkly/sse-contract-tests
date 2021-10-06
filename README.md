# SSE Contract Tests

Implementations of the [Server-Sent Events](https://en.wikipedia.org/wiki/Server-sent_events) protocol must be able to handle many variations of data ordering, network behavior, etc. Providing thorough test coverage of these can be tedious. This project provides a test harness mechanism for running a standardized test suite against any SSE implementation.

This tool was developed to facilitate cross-platform testing of the LaunchDarkly SDKs, all of which rely on SSE for core functionality. However, it can be used for any SSE implementation. At a minimum, it will verify behaviors that are part of the canonical SSE [specification](https://html.spec.whatwg.org/multipage/server-sent-events.html); it can optionally also verify extended capabilities that some SSE implementations provide.

To use this tool, you must first implement a small web service that exercises the features of your SSE implementation. The behavior of the service endpoints is described below. After starting the service, run the test harness and give it the base URL of the test service. The test harness will start its own HTTP server instances to provide SSE data, which it will then ask the test service to connect to and read from.

## Test harness command line

```shell
./sse-contract-tests --url <test service base URL> [other options]

# other options:
#   --port <PORT>  sets the callback port that test services will connect to (default: 8111)
#   --debug        enables verbose logging of test status
```

## Test service endpoints

### Status resource: `GET /`

This resource should return a 200 status to indicate that the service has started. Optionally, it can also return a JSON object in the response body, with the following properties:

* `capabilities`: An array of strings describing optional features that this SSE implementation supports:
  * `"comments"`: The SSE client allows the caller to read comment lines. All SSE implementations must be able to handle comment lines, but many of them will simply discard the comments and not allow them to be seen.
  * `"cr-only"`: The SSE client is able to recognize a single CR (0x0D) as a line terminator. The SSE spec allows CR, LF, or CRLF, but some implementations are not fully compliant and only allow LF and CRLF.
  * `"headers"`: The SSE client can be configured to send custom headers.
  * `"last-event-id"`: The SSE client can be configured to send a specific `Last-Event-Id` value in its initial HTTP request.
  * `"post"`: The SSE client can be configured to send a `POST` request with a body instead of a `GET`.
  * `"report"`: The SSE client can be configured to send a `REPORT` request with a body instead of a `GET`.

The test harness will use the `capabilities` information to decide whether to run optional parts of the test suite that relate to those capabilities.

### Stream resource: `POST /`

A `POST` request indicates that the test suite wants to start an instance of the SSE client. The request body is a JSON object with the following properties. All of the properties except `url` are optional.

* `url`: The URL of an SSE endpoint created by the test suite.
* `tag`: A string describing the current test, if desired for logging.
* `initialDelayMs`: An optional integer specifying the initial reconnection delay parameter, in milliseconds. Not all SSE client implementations allow this to be configured, but the test harness will send a value anyway in an attempt to avoid having reconnection tests run unnecessarily slowly.
* `lastEventId`: An optional string which should be sent as the `Last-Event-Id` header in the initial HTTP request. The test suite will only set this property if the test service has the `"last-event-id"` capability.
* `body`: A string specifying data to be sent in the HTTP request body. The test suite will only set this property if the test service has the `"post"` or `"report"` capability.
* `headers`: A JSON object containing additional HTTP header names and string values. The SSE client should be configured to add these headers to its HTTP requests. The test suite will only set this property if the test service has the `"headers"` capability. Header names can be assumed to all be lowercase.
* `method`: A string specifying an HTTP method to use instead of `GET`. The test suite will only set this property if the test service has the `"post"` or `"report"` capability.

The response to this request is a streaming response using chunked transfer encoding. This is not an SSE stream, since then the test suite would have to rely on a specific SSE implementation to verify a different SSE implementation. Instead, it uses a simpler format where each "message" is a JSON object followed by a single LF (`\n`). The JSON object itself cannot contain any unescaped LF characters.

The JSON message can be one of the following:

#### `hello` message

The test service must send this message first in each response. This just tells the test harness that the stream is being created, regardless of whether it succeeds or fails.

```json
{ "kind": "hello" }
```

#### `event` message

This message indicates that the test service has received an event from the SSE stream. The `type`, `data`, `id`, and `retry` fields correspond to the fields of an SSE event. All but `data` are optional.

```json
{
  "kind": "event",
  "event": {
    "type": "put",
    "data": "my-event-data",
    "id": "my-event-ID",
    "retry": 1000
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
