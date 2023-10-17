build: wallhaven_dl

wallhaven_dl: $(shell find . -name '*.go')
	go build -o wallhaven_dl .

run:
	go run main.go

tidy:
	go mod tidy

clean:
	rm -f wallhaven_dl

uninstall:
	rm -f /usr/bin/wallhaven_dl

install:
	cp wallhaven_dl /usr/bin
