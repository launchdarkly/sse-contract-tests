# Optional SSE features

The core SSE specification was designed to support basic stream functionality in web applications, as defined by the `EventSource` API that is available in JavaScript in most browsers. However, for more complex use cases such as LaunchDarkly, it's desirable to have more sophisticated control over the SSE client. LaunchDarkly's SSE implementations therefore provide some additional features, whose behavior is described below.

By default, the test harness will not run tests for these features, since an SSE implementation can be fully valid without including them. Tests for these are enabled based on the `capabilities` list returned by the test service. If the capability string for a particular features is in that list, then the test harness will run tests for it and will expect it to behave as described below.

For more about the `capabilities` list, properties that the test harness can set in the client configuration, commands that the test harness may send, and callback messages that the test harness may expect, see [SSE test service specification](./service_spec.md).

## Reading comments (capability: `"comments"`)

This means that the SSE client can report comment lines to the caller.

In the SSE specification, a "comment" is defined as a line of text that begins with a colon. The protocol does not define any action to be taken for comment lines; parsers can simply discard them. However, in a particular application they might be useful as metadata that is not tied to a specific event.

If this capability is enabled, the test harness will expect to receive a `"comment"` callback message whenever the client has read a comment line (regardless of whether it has parsed an event). The comment text in the message should not include the leading colon.

Note that the syntax rules in SSE specification show "comment" as part of an "event" block, and "event" is always terminated by two line endings. But, since the core spec does not define any action at all to be taken for comments, that does not mean a client with this capability needs to wait for a double line break before reporting a comment. It simply means that syntactically, any number of comment lines _can_ appear wherever an event field could appear, and that the client should not report an event until it has fully parsed an event.

## Type-specific listeners (capability: `"event-type-listeners"`)

This means that the SSE client's API requires the caller to explicitly listen for any event type that is not the default `"message"`.

The `EventSource` browser API, which is included in the SSE specification, uses a JavaScript event model. The event name is equal to the `type:` field of the event, or `message` if not specified. That means that in order for an application to listen for events (and in order for the test harness to do a test for receiving events), it must specify what type of events it wants.

However, SSE implementations do not need to imitate the `EventSource` API in that regard; the choice of how to listen for or dispatch events can be done in many ways, using idiomatic patterns for different languages. In most implementations, it is possible to simply listen for events in general and not have to specify the type ahead of time. The test harness assumes that that is the case unless otherwise specified.

If this capability is enabled, the test harness will send a `"listen"` command for a specific event type whenever it intends to do a test that uses that type; otherwise no such command will be sent. This is defined as an opt-in behavior so that most test services do not need to provide a no-op implementation for the `"listen"` command.

## Custom headers (capability: `"headers"`)

This means that the caller can tell the SSE client to add arbitrary headers to its HTTP requests.

If this capability is enabled, the test harness will expect that any custom headers it specifies in the `headers` property of the client configuration will be copied into the HTTP request(s) made by the client.

The SSE specification itself makes no requirements about headers except that caching must be disabled, and that if the client is a web browser it must provide any necessary CORS headers for cross-origin requests. SSE clients _may_ add `Accept: text/event-stream`, but it is not required; the basic test suite will only verify that _if_ there is an `Accept` header, it allows the `text/event-stream` type (since SSE servers must always use that type).

## Initial last event ID (capability `"last-event-id"`)

This means that the caller can specify the value for the `Last-Event-Id` header in the client's initial HTTP request.

Implementations of the `EventSource` API in browsers always start with no `Last-Event-Id`, and set it only when reconnecting if there was a non-empty `id:` field in a previous event. However, with SSE implementations that are not `EventSource`, an application may wish to preset `Last-Event-Id` based on information it received in some other way.

If this capability is enabled, the test harness will expect that any non-null and non-empty string it specifies in the `lastEventId` property of the client configuration will be copied into the `Last-Event-Id` header in the client's initial request, and treated as if it had been received as an `id:` in a previous event.

## Sending a POST request (capability `"post"`)

This means that the caller can tell the SSE client to send a `POST` request instead of a `GET` request, and specify the request body.

If this capability is enabled, the test harness will expect that it can set `method` to `"POST"` and `body` to any string value in the client configuration, and the SSE client will use that method and body. If the test harness also sets a `Content-Type` header in `headers`, the client should use that type, otherwise it can use `text/plain`.

## Setting a read timeout (capability `"read-timeout"`)

This means that the caller can tell the SSE client to drop and restart any connection that receives no data within a given interval.

A read timeout, also called a socket timeout, causes a connection to be treated as failed (returning an I/O error on reads) if that amount of time has elapsed without receiving any new data. Receiving any bytes at all should reset the timeout clock; the received data does not have to be a complete SSE event or a complete line of text.

This may be desirable to mitigate silent connection failures. When a TCP connection is broken without being cleanly shut down (either because network connectivity is lost, or because the process on one end died unexpectedly), it may appear to still be alive. To avoid a condition where an SSE client continues listening forever on a failed connection, applications may want to set a read timeout. The server side can be designed to send arbitrary data, such as an empty comment line, at intervals as a heartbeat to prevent unnecessary disconnects.

If this capability is enabled, the test harness will expect that it can set `readTimeoutMs` to a positive integer value in the client configuration, and the SSE client will set the read timeout to that number of milliseconds. The test harness will expect to see the client drop and retry the connection if the test harness sends no data in that amount of time.

## Sending a REPORT request (capability `"report"`)

This means that the caller can tell the SSE client to send a `REPORT` request instead of a `GET` request, and specify the request body. `REPORT` is a nonstandard method that is basically a cacheable `POST` that is cached based on both the URL and the body, used when it is desirable to put query parameters in the request body instead of in the URL.

If this capability is enabled, the test harness will expect that it can set `method` to `"REPORT"` and `body` to any string value in the client configuration, and the SSE client will use that method and body. If the test harness also sets a `Content-Type` header in `headers`, the client should use that type, otherwise it can use `text/plain`.

## Explicitly restarting the stream (capability `"restart"`)

This means that the caller can tell the SSE client to immediately disconnect the active stream and restart the connection, exactly as it would do if the server had dropped the connection.

If this capability is enabled, the test harness will expect that it can send a `"restart"` command and the client will restart the connection (with the same URL as before) as soon as possible.
