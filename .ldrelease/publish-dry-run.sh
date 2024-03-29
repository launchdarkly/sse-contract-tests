#!/bin/bash

set -eu

# Note that Docker commands in this script are being sudo'd. That's because we are
# already running inside a container, and rather than trying to run a whole nested
# Docker daemon inside that container, we are sharing the host's Docker daemon. But
# the mechanism for doing so involves sharing a socket path (docker.sock) that is
# only accessible by root. Also, the PATH=$PATH is necessary to make sure commands
# like "go" can be found in the sudo environment.

sudo PATH=$PATH make build-release
cp dist/*.tar.gz dist/*.zip "${LD_RELEASE_ARTIFACTS_DIR}"

# Copy the Docker image that goreleaser just built into the artifacts - we only do
# this in a dry run, because in a real release the image will be available from
# DockerHub anyway so there's no point in attaching it to the release.
image_archive_name=sse-contract-tests_docker-image.tar.gz
sudo docker save ldcircleci/sse-contract-tests:${LD_RELEASE_VERSION} | gzip >${LD_RELEASE_ARTIFACTS_DIR}/${image_archive_name}
