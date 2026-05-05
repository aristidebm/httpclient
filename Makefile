.PHONY: build test lint clean format install

build:
	go build -o build/http .

install: build
	cp ./build/http ~/go/bin/

test:
	go test ./...

lint:
	@go vet ./...

clean:
	@rm -f build/http

tidy:
	@go mod tidy

format:
	@go fmt ./...

run:
	@go run ./...
