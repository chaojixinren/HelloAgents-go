.PHONY: all build test lint fmt vet check clean cover

all: check build

build:
	go build -o bin/helloagents ./cmd/helloagents

test:
	go test ./...

test-v:
	go test -v ./...

cover:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html

lint: fmt vet

fmt:
	@test -z "$$(gofmt -l .)" || (echo "gofmt check failed:" && gofmt -l . && exit 1)

vet:
	go vet ./...

check: fmt vet test

clean:
	rm -f bin/helloagents coverage.out coverage.html

doctor:
	go run ./cmd/helloagents doctor
