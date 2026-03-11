.PHONY: \
	build \
	build-server \
	build-agent \
	clean \
	test \
	run-server \
	run-server-env \
	run-agent \
	run-agent-env

SERVER=cmd/server/server
AGENT=cmd/agent/agent

SERVER_ADDRESS_DEFAULT=localhost:8080
SERVER_ADDRESS_ENV=localhost:8088

build: build-server build-agent

build-server:
	go build -o $(SERVER) ./cmd/server

build-agent:
	go build -o $(AGENT) ./cmd/agent

clean:
	rm -f $(SERVER) $(AGENT)

test:
	go test ./...

run-server: build-server
	./$(SERVER) -a=$(SERVER_ADDRESS_DEFAULT)

run-server-env: build-server
	ADDRESS=$(SERVER_ADDRESS_ENV) ./$(SERVER)

run-agent: build-agent
	./$(AGENT) -a=$(SERVER_ADDRESS_DEFAULT) -p=2 -r=10

run-agent-env: build-agent
	ADDRESS=$(SERVER_ADDRESS_ENV) POLL_INTERVAL=5 REPORT_INTERVAL=15 ./$(AGENT)
