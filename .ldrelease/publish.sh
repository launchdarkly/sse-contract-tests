#!/bin/bash

set -eu

# Note that Docker commands in this script are being sudo'd. That's because we are
# already running inside a container, and rather than trying to run a whole nested
# Docker daemon inside that container, we are sharing the host's Docker daemon. But
# the mechanism for doing so involves sharing a socket path (docker.sock) that is
# only accessible by root.

docker_username="$(cat "${LD_RELEASE_SECRETS_DIR}/docker_username")"
cat "${LD_RELEASE_SECRETS_DIR}/docker_token" | sudo docker login --username "${docker_username}" --password-stdin

sudo make publish-release
cp dist/*.tar.gz dist/*.zip "${LD_RELEASE_ARTIFACTS_DIR}"
