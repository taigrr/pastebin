.PHONY: build install test clean

all: build

build:
	go build -o pastebin .
	go build -o pb ./cmd/pb/

install:
	go install .
	go install ./cmd/pb/

test:
	go test -v -cover -coverprofile=coverage.txt -covermode=atomic -coverpkg=./... -race ./...

clean:
	rm -f pastebin pb coverage.txt
