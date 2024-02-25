build: wallhaven_dl

wallhaven_dl: $(shell find . -name '*.go')
	go build -o dist/ .

run: build
	./dist/wallhaven_dl

tidy:
	go mod tidy

clean:
	rm -rf dist

uninstall:
	rm -f /usr/bin/wallhaven_dl
	rm -f /usr/share/zsh/site-functions/_wallhaven_dl

install:
	cp ./dist/wallhaven_dl /usr/bin
	cp ./completions/_wallhaven_dl /usr/share/zsh/site-functions/_wallhaven_dl
