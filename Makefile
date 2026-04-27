.PHONY: build test lint clean format install

build:
	go build -o build/http .

test:
	go test ./...

lint:
	@go vet ./...

clean:
	@rm -f build/http

install:
	@go mod tidy

format:
	@go fmt ./...

run:
	@go run ./...
