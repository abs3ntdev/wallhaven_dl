BINARY := wallhaven_dl
DIST := dist
PREFIX ?= /usr

.PHONY: build run test vet tidy clean install uninstall

build:
	go build -o $(DIST)/$(BINARY) .

run: build
	./$(DIST)/$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(DIST)

install: build
	install -Dm755 $(DIST)/$(BINARY) $(DESTDIR)$(PREFIX)/bin/$(BINARY)
	install -Dm644 completions/_$(BINARY) $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_$(BINARY)

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/$(BINARY)
	rm -f $(DESTDIR)$(PREFIX)/share/zsh/site-functions/_$(BINARY)
