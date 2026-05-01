.PHONY: build clean

BINDIR := ./bin
BINARY := keysync

build:
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/$(BINARY) ./cmd/keysync

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
