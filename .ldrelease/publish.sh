#!/bin/bash

docker_username="$(cat "${LD_RELEASE_SECRETS_DIR}/docker_username")"
cat "${LD_RELEASE_SECRETS_DIR}/docker_token" | sudo docker login --username "${docker_username}" --password-stdin

make publish-release
