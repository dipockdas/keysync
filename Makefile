.PHONY: build clean sign build-signed run test test-short test-platform test-clients test-all release distclean bump-formula

BINDIR  := ./bin
BINARY  := keysync
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/dipockdas/keysync/internal/commands.Version=$(VERSION)

build:
	@mkdir -p $(BINDIR)
	go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$(BINARY) ./cmd/keysync

# Sign the binary with a Developer ID certificate for macOS Keychain access control.
# Once signed, "Always Allow" in keychain dialogs persists across rebuilds.
# Usage: make build && make sign
sign:
	@if [ "$(shell uname)" != "Darwin" ]; then \
		echo "signing is only available on macOS"; exit 1; \
	fi
	@identity=$$(security find-identity -v -p basic 2>/dev/null | grep "Developer ID Application" | head -1 | sed 's/.*"\(.*\)"/\1/'); \
	if [ -z "$$identity" ]; then \
		echo "No Developer ID Application certificate found."; \
		echo "Run: security find-identity -v -p basic"; \
		exit 1; \
	fi; \
	echo "Signing with: $$identity"; \
	codesign --sign "$$identity" \
		--options runtime \
		--timestamp \
		--force \
		$(BINDIR)/$(BINARY); \
	codesign -dvvv $(BINDIR)/$(BINARY) 2>&1 | grep -E '^Signed|^Authority|^TeamIdentifier'

# Build and sign in one step (macOS only; falls back to plain build on other platforms).
build-signed: build
	@if [ "$(shell uname)" = "Darwin" ]; then \
		$(MAKE) sign; \
	fi

# Install signed binary to ~/.local/bin (macOS). Run keysync trust after first install.
install-signed: build-signed
	@mkdir -p $(HOME)/.local/bin
	install -m 755 $(BINDIR)/$(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed $(HOME)/.local/bin/$(BINARY) — run: keysync trust"

clean:
	rm -rf $(BINDIR)

run: build
	$(BINDIR)/$(BINARY) $(ARGS)

test:
	go test ./internal/... -v -race -count=1

test-short:
	go test ./internal/... -race -count=1

test-platform:
	go test ./internal/platforms/... -v -count=1

test-clients:
	cd clients/go && go test ./... -v -race -count=1

test-all: test test-clients
	@echo "All tests passed."

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

# Regenerate Formula/keysync.rb from the latest (or given) GitHub release.
# Usage: make bump-formula TAG=v1.0.3
bump-formula:
	@./scripts/update-homebrew-formula.sh $(TAG)
