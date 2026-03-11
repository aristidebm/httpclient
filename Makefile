.PHONY: build test lint clean

build:
	go build -o build/http .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f build/http 

tidy:
	go mod tidy
