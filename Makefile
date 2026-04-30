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
