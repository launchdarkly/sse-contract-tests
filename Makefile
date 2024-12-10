GORELEASER_VERSION=v0.175.0
GORELEASER_CMD=curl -sL https://git.io/goreleaser | GOPATH=$(mktemp -d) VERSION=$(GORELEASER_VERSION) bash -s -- --rm-dist

GOLANGCI_LINT_VERSION=v1.43.0
LINTER=./bin/golangci-lint
LINTER_VERSION_FILE=./bin/.golangci-lint-version-$(GOLANGCI_LINT_VERSION)

.PHONY: build clean test lint build-release publish-release

build:
	go build

clean:
	go clean

test:
	go test ./...

$(LINTER_VERSION_FILE):
	rm -f $(LINTER)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b ./bin $(GOLANGCI_LINT_VERSION)
	touch $(LINTER_VERSION_FILE)

lint: $(LINTER_VERSION_FILE)
	$(LINTER) run ./...

build-release:
	$(GORELEASER_CMD) --snapshot --skip-publish --skip-validate

publish-release:
	$(GORELEASER_CMD)
