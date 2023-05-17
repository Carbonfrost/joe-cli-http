-include eng/Makefile

.DEFAULT_GOAL = build
.PHONY: \
	generate \
	watch \
	lint \
	install \
	-install-% \
	coverage \
	coveragereport \

BUILD_VERSION=$(shell git rev-parse --short HEAD)
GO_LDFLAGS=-X 'github.com/Carbonfrost/joe-cli-http/internal/build.Version=$(BUILD_VERSION)'

build: generate

watch:
	@ find Makefile . -name '*.go' | entr -c cli --version --plus --time generate

generate:
	$(Q) $(OUTPUT_COLLAPSED) go generate ./...

lint:
	$(Q) go run honnef.co/go/tools/cmd/staticcheck -checks 'all,-ST*' $(shell go list ./...)

install: -install-wig -install-toupee -install-weave

-install-%: build -check-env-PREFIX -check-env-_GO_OUTPUT_DIR
	$(Q) eng/install "${_GO_OUTPUT_DIR}/$*" $(PREFIX)/bin

coverage:
	$(Q) go test -coverprofile=coverage.txt -covermode=atomic ./...

coveragereport: coverage
	$(Q) go tool cover -html=coverage.txt
