# How to deploy this tool

In a CI job for an SSE implementatin, the most convenient way to run the test suite is by invoking the `downloader/run.sh` script, which downloads the compiled executable and runs it. You can download this script directly from GitHub and pipe it to `bash` or `sh`. This is similar to how tools such as Goreleaser are normally run. You must set `VERSION` to the desired version of the tool, and `PARAMS` to the command-line parameters.

```shell
curl -s https://raw.githubusercontent.com/launchdarkly/sse-test-harness/master/downloader/run.sh \
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
