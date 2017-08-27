export GO15VENDOREXPERIMENT=1

BENCH_FLAGS ?= -cpuprofile=cpu.pprof -memprofile=mem.pprof -benchmem
PKGS ?= $(shell go list ./... | grep -v /vendor/)
# Many Go tools take file globs or directories as arguments instead of packages.
PKG_FILES ?= $(shell find . -path ./vendor -prune -o -name '*.go' -print)

# The linting tools evolve with each Go version, so run them only on the latest
# stable release.
GO_VERSION := $(shell go version | cut -d " " -f 3)
GO_MINOR_VERSION := $(word 2,$(subst ., ,$(GO_VERSION)))


.PHONY: all
all: test

.PHONY: dependencies
dependencies:
	@echo "Installing dep and locked dependencies..."
	go get -u github.com/golang/dep/cmd/dep
	dep ensure
	@echo "Installing test dependencies..."
	go get -u github.com/axw/gocov/gocov
ifdef SHOULD_LINT
	@echo "Installing golint..."
	go get -u github.com/golang/lint/golint
else
	@echo "Not installing golint, since we don't expect to lint on" $(GO_VERSION)
endif

# Disable printf-like invocation checking due to testify.assert.Error()
VET_RULES := -printf=false

.PHONY: test
test:
	go test -race $(PKGS)

.PHONY: cover
cover:
	./scripts/cover.sh $(PKGS)

.PHONY: bench
BENCH ?= .
bench:
	@$(foreach pkg,$(PKGS),go test -bench=$(BENCH) -run="^$$" $(BENCH_FLAGS) $(pkg);)
