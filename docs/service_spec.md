# SSE test service specification

## Service endpoints

### Status resource: `GET /`

This resource should return a 200 status to indicate that the service has started. Optionally, it can also return a JSON object in the response body, with the following properties:

* `capabilities`: An array of strings describing optional features that this SSE implementation supports. See: [Optional SSE features](optional_features.md)

The test harness will use the `capabilities` information to decide whether to run optional parts of the test suite that relate to those capabilities.

### Stop test service: `DELETE /`

The test harness sends this request at the end of a test run if you have specified `--stop-service-at-end`. The test service should simply quit. This is a convenience so CI scripts can simply start the test service in the background and assume it will be stopped for them.

### Create stream: `POST /`

A `POST` request indicates that the test harness wants to start an instance of the SSE client. The request body is a JSON object with the following properties. All of the properties except `streamUrl` and `callbackUrl` are optional.

* `streamUrl`: The URL of an SSE endpoint created by the test harness.
* `callbackUrl`: The base URL of a callback endpoint created by the test harness (see "Callback endpoint" below).
* `tag`: A string describing the current test, if desired for logging.
* `initialDelayMs`: An optional integer specifying the initial reconnection delay parameter, in milliseconds. Not all SSE client implementations allow this to be configured, but the test harness will send a value anyway in an attempt to avoid having reconnection tests run unnecessarily slowly.
* `readTimeoutMs`: An optional integer specifying the desired read timeout/socket timeout, in milliseconds. The test harness will only set this property if the test service has the `"read-timeout"` capability.
* `lastEventId`: An optional string which should be sent as the `Last-Event-Id` header in the initial HTTP request. The test harness will only set this property if the test service has the `"last-event-id"` capability.
* `headers`: A JSON object containing additional HTTP header names and string values. The SSE client should be configured to add these headers to its HTTP requests. The test harness will only set this property if the test service has the `"headers"` capability. Header names can be assumed to all be lowercase.
* `method`: A string specifying an HTTP method to use instead of `GET`. The test harness will only set this property if the test service has the `"post"` or `"report"` capability.
* `body`: A string specifying data to be sent in the HTTP request body. The test harness will only set this property if the test service has the `"post"` or `"report"` capability.

The response to a valid request is any HTTP `2xx` status, with a `Location` header whose value is the URL of the test service resource representing this instance (that is, the one that would be used for "Close stream" or "Send command" as described below).

If any parameters are invalid, return HTTP `400`.

### Send command: `POST <URL of stream instance>`

A `POST` request to the resource that was returned by "Create stream" means the test harness wants to do something to an existing SSE client instance. The request body is a JSON object which can be one of the following:

#### `listen` command

```json
{
  "command": "listen",
  "listen": {
    "type": "<EVENT TYPE>"
  }
}
```

This means the SSE client should be ready to receive events with the type `EVENT TYPE`. This will only be sent if the test service has the `"event-type-listeners"` capability.

#### `restart` command

```json
{
  "command": "restart"
}
```

This means SSE client should disconnect and reconnect with the same stream URL. This will only be sent if the test service has the `"restart"` capability.

Return any HTTP `2xx` status, `400` for an unrecognized command, or `404` if there is no such stream.

If the SSE implementation does not support any special commands, then the test service doesn't need to implement this endpoint.

### Close stream: `DELETE <URL of stream instance>`

A `DELETE` request to the resource that was returned by "Create stream" means the test harness is done with this SSE client instance and the test service should stop it.

Return any HTTP `2xx` status, or `404` if there is no such stream.

## Callback endpoint

When the test harness tells the test service to create a stream, it provides a callback URL that is specific to that stream. The test service will use this to deliver information about the status of the stream via POST requests.

To avoid race conditions where the test harness might process messages asynchronously in the wrong order, the test service must maintain a callback message counter for each stream, starting at 1 for the first callback it sends. Add this to the URL path: for instance, if the base callback URL is `http://testservice:8111/endpoints/99`, callbacks should be sent to `http://testservice:8111/endpoints/99/1`, `http://testservice:8111/endpoints/99/2`, etc.

The request body is always a JSON object, which can be one of the following:

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
