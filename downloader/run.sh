#!/bin/sh

set -e

# Downloads some version of the sse-contract-tests command, from the compiled binaries that are
# published to GitHub, and runs it. You must specify either a full version string (v1.2.3)
# or a partial version (v1) in the environment variable VERSION, and any parameters you want to
# pass to the command in PARAMS.
#
# Sometimes you will hit GitHub API rate limits when running this command. If you have a GitHub
# token, pass it in the environment variable GITHUB_TOKEN.
#
# This script can be used in Linux, MacOS, or Windows (MSYS/MINGW/CYGWIN). It will download
# whichever binary is appropriate for the current OS and architecture. It requires /bin/sh and
# the commands "grep", "sed", and "curl". On Windows it also requires "unzip"; on other
# platforms it requires "tar".

RELEASES_API_URL=https://api.github.com/repos/launchdarkly/sse-contract-tests/releases
RELEASES_SITE_URL=https://github.com/launchdarkly/sse-contract-tests/releases

# Detect OS
case "$(uname -s)" in
  Linux*)     OS_TYPE=Linux;;
  Darwin*)    OS_TYPE=Darwin;;
  CYGWIN*)    OS_TYPE=Windows;;
  MINGW*)     OS_TYPE=Windows;;
  MSYS_NT*)   OS_TYPE=Windows;;
  *)          OS_TYPE="UNKNOWN"
esac

if [ "${OS_TYPE}" = "UNKNOWN" ]; then
  echo "Unrecognized or unsupported operating system '$(uname -s)'." >&2
  echo "Supported: Linux, macOS (Darwin), Windows (MSYS, MINGW, CYGWIN)." >&2
  exit 1
fi

# Detect architecture and normalize
ARCH=$(uname -m)
if [ "${ARCH}" = "aarch64" ]; then
  ARCH="arm64"
fi

echo "Platform OS:   ${OS_TYPE}" >&2
echo "Platform arch: ${ARCH}" >&2

# Determine archive extension based on OS
case "${OS_TYPE}" in
  Windows) EXTENSION="zip" ;;
  *)       EXTENSION="tar.gz" ;;
esac

EXECUTABLE_ARCHIVE_NAME="sse-contract-tests_${OS_TYPE}_${ARCH}.${EXTENSION}"
echo "Archive name:  ${EXECUTABLE_ARCHIVE_NAME}" >&2

# Set up auth header if a GitHub token is available. This avoids API rate limiting
# on shared CI runners.
if [ -n "${GITHUB_TOKEN}" ]; then
  AUTH_HEADER="Authorization: token ${GITHUB_TOKEN}"
  echo "GitHub token:  provided" >&2
else
  AUTH_HEADER=""
  echo "GitHub token:  not provided (API requests may be rate-limited)" >&2
fi

if [ -z "${VERSION}" -o -z "${PARAMS}" ]; then
  echo 'You must specify a version string in $VERSION and command parameters in $PARAMS' >&2
  exit 1
fi

# Log a message to stderr so it doesn't interfere with command substitution captures.
log() {
  echo "$@" >&2
}

# Perform an HTTP request with curl, logging important details about the response.
# All logging goes to stderr. If no output file is specified, the response body is
# written to stdout (so callers can capture it).
# Usage: do_curl [-o output_file] <url>
do_curl() {
  OUTPUT_FILE=""
  if [ "$1" = "-o" ]; then
    OUTPUT_FILE="$2"
    shift 2
  fi
  URL="$1"

  log ""
  log "HTTP request:  GET ${URL}"

  HEADER_FILE=$(mktemp)

  CURL_ARGS="--fail -s -L -D ${HEADER_FILE}"
  if [ -n "${AUTH_HEADER}" ]; then
    CURL_ARGS="${CURL_ARGS} -H '${AUTH_HEADER}'"
  fi

  if [ -n "${OUTPUT_FILE}" ]; then
    CURL_ARGS="${CURL_ARGS} -o '${OUTPUT_FILE}'"
  fi

  CURL_CMD="curl ${CURL_ARGS} '${URL}'"

  RESPONSE=""
  CURL_EXIT=0
  if [ -n "${OUTPUT_FILE}" ]; then
    eval "${CURL_CMD}" || CURL_EXIT=$?
  else
    RESPONSE=$(eval "${CURL_CMD}") || CURL_EXIT=$?
  fi

  # Log response details from headers
  if [ -f "${HEADER_FILE}" ]; then
    STATUS_LINE=$(head -n 1 "${HEADER_FILE}" | tr -d '\r')
    log "HTTP response: ${STATUS_LINE}"

    CONTENT_TYPE=$(grep -i '^content-type:' "${HEADER_FILE}" | tail -n 1 | tr -d '\r' | sed 's/^[^:]*: *//')
    if [ -n "${CONTENT_TYPE}" ]; then
      log "Content-Type:  ${CONTENT_TYPE}"
    fi

    RATE_LIMIT=$(grep -i '^x-ratelimit-remaining:' "${HEADER_FILE}" | tail -n 1 | tr -d '\r' | sed 's/^[^:]*: *//')
    if [ -n "${RATE_LIMIT}" ]; then
      log "Rate limit remaining: ${RATE_LIMIT}"
    fi

    rm -f "${HEADER_FILE}"
  fi

  if [ "${CURL_EXIT}" -ne 0 ]; then
    log "HTTP request failed with curl exit code ${CURL_EXIT}"
    return "${CURL_EXIT}"
  fi

  if [ -z "${OUTPUT_FILE}" ]; then
    echo "${RESPONSE}"
  fi
}

resolve_version() {
  if echo "$1" | grep -q '^v[^.][^.]*\.[^.][^.]*\.'; then
    # It's already a complete version string
    log "Resolved version: $1 (already complete)"
    echo "$1"
    return
  fi

  log ""
  log "Resolving partial version '${1}' from GitHub releases API..."

  API_RESPONSE=$(do_curl "${RELEASES_API_URL}")

  ALL_VERSIONS=$(echo "${API_RESPONSE}" \
    | grep "tag_name" \
    | sed -e 's/.*:[^"]*"\([^"]*\).*/\1/')

  log ""
  log "Available versions:"
  echo "${ALL_VERSIONS}" | sed 's/^/  /' >&2

  RESOLVED=$(echo "${ALL_VERSIONS}" \
    | grep "^$1\." \
    | head -n 1)

  if [ -n "${RESOLVED}" ]; then
    log ""
    log "Resolved version: ${RESOLVED}"
  fi

  echo "${RESOLVED}"
}

VERSION_TO_DOWNLOAD=$(resolve_version "${VERSION}")
if [ -z "${VERSION_TO_DOWNLOAD}" ]; then
  echo "" >&2
  echo "Unable to find a release matching '${VERSION}'" >&2
  exit 1
fi

TEMP_DIR="/tmp/sse-contract-tests_${VERSION_TO_DOWNLOAD}"
EXECUTABLE="${TEMP_DIR}/sse-contract-tests"
DOWNLOAD_URL="${RELEASES_SITE_URL}/download/${VERSION_TO_DOWNLOAD}/${EXECUTABLE_ARCHIVE_NAME}"

log ""
log "Download name: ${EXECUTABLE_ARCHIVE_NAME}"

if [ ! -x "${EXECUTABLE}" ]; then
  rm -rf "${TEMP_DIR}"
  mkdir "${TEMP_DIR}"
  do_curl -o "${TEMP_DIR}/archive.${EXTENSION}" "${DOWNLOAD_URL}"
  if [ "${EXTENSION}" = "zip" ]; then
    unzip -o "${TEMP_DIR}/archive.${EXTENSION}" -d "${TEMP_DIR}"
  else
    tar -xf "${TEMP_DIR}/archive.${EXTENSION}" -C "${TEMP_DIR}"
  fi
fi

log ""
sh -c "${EXECUTABLE} $PARAMS"
