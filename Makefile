.PHONY: build clean run test test-short test-platform release distclean

BINDIR  := ./bin
BINARY  := keysync
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/dipockdas/keysync/internal/commands.Version=$(VERSION)

build:
	@mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BINARY) ./cmd/keysync

clean:
	rm -rf $(BINDIR)

run: build
	$(BINDIR)/$(BINARY) $(ARGS)

test:
	go test ./internal/store/... ./internal/config/... ./internal/crypto/... ./internal/commands/... -v -race -count=1

test-short:
	go test ./internal/store/... ./internal/config/... ./internal/crypto/... ./internal/commands/... -race -count=1

test-platform:
	go test ./internal/platforms/... -v -count=1

# Cross-compile for all supported platforms.
# Usage: make release VERSION=v0.1.0
release: distclean
	@mkdir -p dist
	$(call xbuild,darwin,amd64)
	$(call xbuild,darwin,arm64)
	$(call xbuild,linux,amd64)
	$(call xbuild,linux,arm64)
	$(call xbuild,windows,amd64)
	$(call xbuild,windows,arm64)
	@echo "Release artifacts in dist/:"
	@ls -1 dist/

define xbuild
	@echo "Building for $(1)/$(2)..."
	@GOOS=$(1) GOARCH=$(2) go build -ldflags "$(LDFLAGS)" \
		-o dist/keysync_$(1)_$(2)/$(BINARY)$(if $(filter windows,$(1)),.exe,) \
		./cmd/keysync
endef

distclean:
	rm -rf dist
