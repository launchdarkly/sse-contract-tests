
GORELEASER_VERSION=v0.141.0
GORELEASER_CMD=curl -sL https://git.io/goreleaser | GOPATH=$(mktemp -d) VERSION=$(GORELEASER_VERSION) bash -s -- --rm-dist

GOLANGCI_LINT_VERSION=v1.27.0
LINTER=./bin/golangci-lint
LINTER_VERSION_FILE=./bin/.golangci-lint-version-$(GOLANGCI_LINT_VERSION)

.PHONY: build clean lint build-release publish-release docker-build docker-push docker-smoke-test

build:
	go build

clean:
	go clean

$(LINTER_VERSION_FILE):
	rm -f $(LINTER)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)
	touch $(LINTER_VERSION_FILE)

lint: $(LINTER_VERSION_FILE)
	$(LINTER) run ./...

build-release:
	$(GORELEASER_CMD) --snapshot --skip-publish --skip-validate --debug

publish-release:
	$(GORELEASER_CMD)

docker-build:
	docker build --tag ldcircleci/sse-contract-tests .

docker-smoke-test: docker-build
	@# To verify that the built image actually works, we'll run it against a fake service
	@# that's set up to always return a 500 error. Seeing the expected error message from
	@# the test harness proves that the harness did run and connected to the right URL.
	@docker network create sse-contract-tests-shared 2>/dev/null || true

	@echo "Starting fake test service container"
	docker run --rm -d -p 8000:8000 --network sse-contract-tests-shared --name smoke-test-fake-service cimg/base:2021.10 \
		bash -c "while true ; do echo -e \"HTTP/1.1 500 Nope\n\n\" | nc -l -p 8000 ; done"

	@echo "Running test harness against fake service"
	(docker run --rm -p 8111:8111 --network sse-contract-tests-shared ldcircleci/sse-contract-tests \
		--url http://smoke-test-fake-service:8000 2>&1 || true) \
		| tee /tmp/sse-contract-tests-smoketest.log

	@grep "test service returned status code 500" </tmp/sse-contract-tests-smoketest.log >/dev/null \
		|| (echo "Did not see expected output from test harness - smoke test fails"; exit 1)
	@echo && echo "The 500 error above was expected - smoke test passes"

	@docker stop smoke-test-fake-service >/dev/null
