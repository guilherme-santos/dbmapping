all: test

test:
	go test -v -race ./...
