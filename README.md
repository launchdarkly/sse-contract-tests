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
* `--stop-service-at-end` - tells the test service to exit after the test run
* `--debug` - enables verbose logging of test actions for failed tests
* `--debug-all` - enables verbose logging of test actions for all tests

For `--run` and `--skip`, the name of a test is the string that appears between brackets in the test output. This may have multiple segments delimited by slashes, such as `name of test category/name of subtest`.

## Downloading/deploying

In a CI job for an SSE implementation, the most convenient way to run the test suite is by invoking the `downloader/run.sh` script, which downloads the compiled executable and runs it. You can download this script directly from GitHub and pipe it to `bash` or `sh`. This is similar to how tools such as Goreleaser are normally run. You must set `VERSION` to the desired version of the tool, and `PARAMS` to the command-line parameters.

```shell
curl -s https://raw.githubusercontent.com/launchdarkly/sse-contract-tests/master/downloader/run.sh \
  | VERSION=v1 PARAMS="--url http://localhost:8000" sh
```

You can specify an exact version string such as `v1.0.0` in `VERSION`, but it is better to specify only a major version so that you will automatically get any backward-compatible improvements in the test harness.

There is also a published Docker image that you can run, `ldcircleci/sse-contract-tests`. Again you can specify either a specific version or a major version, such as `ldcircleci/sse-contract-tests:1`. In order for this to work, the test harness must be able to see the test service and vice versa, so you must remember to expose the callback port on the test harness container-- and you must tell Docker to use either host networking (if the test service is running locally outside of Docker) or a shared network (if the test service is running in Docker).

```shell
# With host networking
docker run --network host ldcircleci/sse-contract-tests:1 \
  --url http://localhost:8000

# With a shared network
docker run --network my-network-name ldcircleci/sse-contract-tests:1 \
  --url http://test-service-container-name:8000
```

## Test service endpoints

### Status resource: `GET /`

This resource should return a 200 status to indicate that the service has started. Optionally, it can also return a JSON object in the response body, with the following properties:

* `capabilities`: An array of strings describing optional features that this SSE implementation supports:
  * `"comments"`: The SSE client allows the caller to read comment lines. All SSE implementations must be able to handle comment lines, but many of them will simply discard the comments and not allow them to be seen.
  * `"event-type-listeners"`: This means that the SSE client's API requires the caller to explicitly listen for any event type that is not the default `"message"`.
  * `"headers"`: The SSE client can be configured to send custom headers.
  * `"last-event-id"`: The SSE client can be configured to send a specific `Last-Event-Id` value in its initial HTTP request.
  * `"post"`: The SSE client can be configured to send a `POST` request with a body instead of a `GET`.
  * `"read-timeout"`: The SSE client can be configured with a specific read timeout (a.k.a. socket timeout).
  * `"report"`: The SSE client can be configured to send a `REPORT` request with a body instead of a `GET`.
  * `"restart"`: The caller can tell the SSE client at any time to disconnect and retry.

The test harness will use the `capabilities` information to decide whether to run optional parts of the test suite that relate to those capabilities.

### Stop test service: `DELETE /`

The test harness sends this request at the end of a test run if you have specified `--stop-service-at-end`. The test service should simply quit. This is a convenience so CI scripts can simply start the test service in the background and assume it will be stopped for them.

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

A `POST` request to the resource that was returned by "Create stream" means the test harness wants to do something to an existing SSE client instance. The request body is a JSON object which can be one of the following:

* `{ "command": "listen", "type": "<EVENT TYPE>" }` - The SSE client should be ready to receive events with the type `EVENT TYPE`. This will only be sent if the test service has the `"event-type-listeners"` capability.
* `{ "command": "restart" }` - The SSE client should disconnect and reconnect with the same stream URL. This will only be sent if the test service has the `"restart"` capability.

Return any HTTP `2xx` status, `400` for an unrecognized command, or `404` if there is no such stream.

If the SSE implementation does not support any special commands, then the test service doesn't need to implement this endpoint.

### Close stream: `DELETE <URL of stream instance>`

A `DELETE` request to the resource that was returned by "Create stream" means the test harness is done with this SSE client instance and the test service should stop it.

Return any HTTP `2xx` status, or `404` if there is no such stream.

## Callback endpoint

When the test harness tells the test service to create a stream, it provides a callback URL that is specific to that stream. The test service should make `POST` requests to this URL to deliver information about the status of the stream. The request body is always a JSON object, which can be one of the following:

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

# About LaunchDarkly

* LaunchDarkly is a continuous delivery platform that provides feature flags as a service and allows developers to iterate quickly and safely. We allow you to easily flag your features and manage them from the LaunchDarkly dashboard.  With LaunchDarkly, you can:
    * Roll out a new feature to a subset of your users (like a group of users who opt-in to a beta tester group), gathering feedback and bug reports from real-world use cases.
    * Gradually roll out a feature to an increasing percentage of users, and track the effect that the feature has on key metrics (for instance, how likely is a user to complete a purchase if they have feature A versus feature B?).
    * Turn off a feature that you realize is causing performance problems in production, without needing to re-deploy, or even restart the application with a changed configuration file.
    * Grant access to certain features based on user attributes, like payment plan (eg: users on the ‘gold’ plan get access to more features than users in the ‘silver’ plan). Disable parts of your application to facilitate maintenance, without taking everything offline.
* LaunchDarkly provides feature flag SDKs for a wide variety of languages and technologies. Check out [our documentation](https://docs.launchdarkly.com/docs) for a complete list.
* Explore LaunchDarkly
    * [launchdarkly.com](https://www.launchdarkly.com/ "LaunchDarkly Main Website") for more information
    * [docs.launchdarkly.com](https://docs.launchdarkly.com/  "LaunchDarkly Documentation") for our documentation and SDK reference guides
    * [apidocs.launchdarkly.com](https://apidocs.launchdarkly.com/  "LaunchDarkly API Documentation") for our API documentation
    * [launchdarkly.com/blog](https://launchdarkly.com/blog/  "LaunchDarkly Blog Documentation") for the latest product updates
