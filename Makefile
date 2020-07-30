INTEGRATION  := $(shell basename $(shell pwd))
BINARY_NAME   = $(INTEGRATION)
GO_PKGS      := $(shell go list ./... | grep -v "/vendor/")
GO_FILES     := ./src/
DEPS          = github.com/kardianos/govendor
TEST_DEPS     = github.com/axw/gocov/gocov github.com/AlekSi/gocov-xml

all: build

build: clean validate compile

build-linux: clean validate compile-linux

clean:
	@echo "=== $(INTEGRATION) === [ clean ]: removing binaries and coverage file..."
	@rm -rfv bin coverage.xml

deps: tools deps-only

tidy:
	@echo "=== $(INTEGRATION) === [ tidy ]: Tidying up go mod..."
	go mod tidy

tools: check-version
	@echo "=== $(INTEGRATION) === [ tools ]: Installing tools required by the project..."
	go get $(TEST_DEPS)

deps-only:
	@echo "=== $(INTEGRATION) === [ deps ]: Installing package dependencies required by the project..."
	go mod download

validate-only:
	@printf "=== $(INTEGRATION) === [ validate ]: running gofmt... "
# `gofmt` expects files instead of packages. `go fmt` works with
# packages, but forces -l -w flags.
	@OUTPUT="$(shell gofmt -l $(GO_FILES))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Incorrect syntax in the following files:" ;\
		echo "$$OUTPUT" ;\
		exit 1 ;\
	fi
	@printf "=== $(INTEGRATION) === [ validate ]: running golint... "
	@OUTPUT="$(shell golint $(GO_PKGS))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Issues found:" ;\
		echo "$$OUTPUT" ;\
		exit 1 ;\
	fi
	@printf "=== $(INTEGRATION) === [ validate ]: running go vet... "
	@OUTPUT="$(shell go vet $(GO_PKGS))" ;\
	if [ -z "$$OUTPUT" ]; then \
		echo "passed." ;\
	else \
		echo "failed. Issues found:" ;\
		echo "$$OUTPUT" ;\
		exit 1;\
	fi

validate: deps

compile-only: deps-only
	@echo "=== $(INTEGRATION) === [ compile ]: building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(GO_FILES)

compile-only-linux: deps-only
	@echo "=== $(INTEGRATION) === [ compile ]: building $(BINARY_NAME)..."
	@rm -rf bin/
	@env GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME) $(GO_FILES)

compile: compile-only

compile-linux: compile-only-linux

package:
	@echo "=== $(INTEGRATION) === [ package ]: packaging release for $(BINARY_NAME)..."
	@cp nri-apigee* bin/
	@cp README.md bin/
	@cp install.sh bin/
	@tar czf nri-apigee-linux.tar.gz bin/*

test-deps: compile-deps
	@echo "=== $(INTEGRATION) === [ test-deps ]: installing testing dependencies..."
	@go get -v $(TEST_DEPS)

test-only:
	@echo "=== $(INTEGRATION) === [ test ]: running unit tests..."
	@gocov test $(GO_PKGS) | gocov-xml > coverage.xml

test: test-deps test-only

check-version:
ifdef GOOS
ifneq "$(GOOS)" "$(NATIVEOS)"
	$(error GOOS is not $(NATIVEOS). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif
ifdef GOARCH
ifneq "$(GOARCH)" "$(NATIVEARCH)"
	$(error GOARCH variable is not $(NATIVEARCH). Cross-compiling is only allowed for 'clean', 'deps-only' and 'compile-only' targets)
endif
endif

.PHONY: all build clean validate-deps validate-only validate compile-deps compile-only compile test-deps test-only test
