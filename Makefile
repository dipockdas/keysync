.PHONY: build clean test test-short

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
