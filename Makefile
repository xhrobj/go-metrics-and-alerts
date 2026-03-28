.PHONY: \
	build build-server build-agent \
	clean \
	test test-race \
	postgres-up postgres-start postgres-stop postgres-rm postgres-connect \
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

RATE_LIMIT=3
SECRET_KEY=god

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

test-race:
	go test -race ./...

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

postgres-connect:
	docker exec -it metrics-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

run-server: build-server
	./$(SERVER) -a=$(SERVER_ADDRESS_DEFAULT) -d=$(POSTGRES_DSN) -k=$(SECRET_KEY)

run-server-env: build-server
	ADDRESS=$(SERVER_ADDRESS_ENV) DATABASE_DSN=$(POSTGRES_DSN) KEY=$(SECRET_KEY) ./$(SERVER)

run-agent: build-agent
	./$(AGENT) -a=$(SERVER_ADDRESS_DEFAULT) -p=2 -r=10 -l=$(RATE_LIMIT) -k=$(SECRET_KEY)

run-agent-env: build-agent
	ADDRESS=$(SERVER_ADDRESS_ENV) POLL_INTERVAL=5 REPORT_INTERVAL=15 RATE_LIMIT=$(RATE_LIMIT) KEY=$(SECRET_KEY) ./$(AGENT)
