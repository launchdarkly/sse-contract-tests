#!/bin/bash

make build-release

# Copy the Docker image that goreleaser just built into the artifacts - we only do
# this in a dry run, because in a real release the image will be available from
# DockerHub anyway so there's no point in attaching it to the release.
image_archive_name=sse-contract-tests-docker-image.tar.gz
docker save launchdarkly/sse-contract-tests:${LD_RELEASE_VERSION} | gzip >${LD_RELEASE_ARTIFACTS_DIR}/${image_archive_name}
