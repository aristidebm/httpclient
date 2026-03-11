.PHONY: build test lint clean

build:
	go build -o cdapi .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f cdapi

tidy:
	go mod tidy
