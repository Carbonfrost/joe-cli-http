-include eng/Makefile

.DEFAULT_GOAL = build
.PHONY: \
	generate \
	watch \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/joe-cli-http/internal/build.Version=$(BUILD_VERSION)'

build: generate

watch:
	@ find Makefile . -name '*.go' | entr -c cli --version --plus --time generate

generate:
	$(Q) go generate ./...
