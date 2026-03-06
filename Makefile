.PHONY: build build-server build-agent clean test run-server run-agent

build: build-server build-agent

build-server:
	go build -o cmd/server/server ./cmd/server

build-agent:
	go build -o cmd/agent/agent ./cmd/agent

clean:
	rm -f cmd/server/server cmd/agent/agent

test:
	go test ./...

run-server: build-server
	./cmd/server/server -a=localhost:8080

run-agent: build-agent
	./cmd/agent/agent -a=localhost:8080 -p=2 -r=10
