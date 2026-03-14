.PHONY: \
	build build-server build-agent \
	clean \
	test \
	postgres-up postgres-start postgres-stop postgres-rm \
	run-server run-server-env \
	run-agent run-agent-env

POSTGRES_USER=metrics
POSTGRES_PASSWORD=password
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=metricsdb
POSTGRES_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

SERVER_ADDRESS_DEFAULT=localhost:8080
SERVER_ADDRESS_ENV=localhost:8088

SERVER=cmd/server/server
AGENT=cmd/agent/agent

build: build-server build-agent

build-server:
	go build -o $(SERVER) ./cmd/server

build-agent:
	go build -o $(AGENT) ./cmd/agent

clean:
	rm -f $(SERVER) $(AGENT)

test:
	go test ./...

postgres-up:
	docker run --name metrics-postgres \
		-e POSTGRES_USER=$(POSTGRES_USER) \
		-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
		-e POSTGRES_DB=$(POSTGRES_DB) \
		-p $(POSTGRES_PORT):5432 \
		-d postgres:16

postgres-start:
	docker start metrics-postgres

postgres-stop:
	docker stop metrics-postgres

postgres-rm:
	docker rm metrics-postgres

run-server: build-server
	./$(SERVER) -a=$(SERVER_ADDRESS_DEFAULT) -d=$(POSTGRES_DSN)

run-server-env: build-server
	ADDRESS=$(SERVER_ADDRESS_ENV) DATABASE_DSN=$(POSTGRES_DSN) ./$(SERVER)

run-agent: build-agent
	./$(AGENT) -a=$(SERVER_ADDRESS_DEFAULT) -p=2 -r=10

run-agent-env: build-agent
	ADDRESS=$(SERVER_ADDRESS_ENV) POLL_INTERVAL=5 REPORT_INTERVAL=15 ./$(AGENT)
