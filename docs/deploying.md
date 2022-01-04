# How to deploy this tool

In a CI job for an SSE implementation, the most convenient way to run the test suite is by invoking the `downloader/run.sh` script, which downloads the compiled executable and runs it. You can download this script directly from GitHub and pipe it to `bash` or `sh`. This is similar to how tools such as Goreleaser are normally run. You must set `VERSION` to the desired version of the tool, and `PARAMS` to the command-line parameters.

```shell
curl -s https://raw.githubusercontent.com/launchdarkly/sse-test-harness/v2.0.0/downloader/run.sh \
  | VERSION=v2 PARAMS="--url http://localhost:8000" sh
```

In this example, `v2.0.0` is the version of the `run.sh` script to use. If there are any significant changes to the script, there will be a new major version, to ensure that CI jobs pinned to previous versions will not fail.

The `VERSION=v2` setting is what determines what version of the actual tests to use. It's best to specify only a major version so that you will automatically get any backward-compatible improvements in the test harness-- as long as you keep in mind that this might cause a build to fail that previously passed, if a more sensitive test is added (that is, if the test harness now detects a kind of noncompliance with the SSE spec that it did not previously check for). If you want to make sure your builds will never break due to such an improvement in the tests, you can instead pin to a specific version string such as `VERSION=v2.0.0`, but be aware that this could mean a bug is overlooked.

There is also a published Docker image that you can run, `ldcircleci/sse-contract-tests`. Again you can specify either a specific version or a major version, such as `ldcircleci/sse-contract-tests:1`. In order for this to work, the test harness must be able to see the test service and vice versa, so you must remember to expose the callback port on the test harness container-- and you must tell Docker to use either host networking (if the test service is running locally outside of Docker) or a shared network (if the test service is running in Docker).

```shell
# With host networking
docker run --network host ldcircleci/sse-contract-tests:1 \
  --url http://localhost:8000

# With a shared network
docker run --network my-network-name ldcircleci/sse-contract-tests:1 \
  --url http://test-service-container-name:8000
```
