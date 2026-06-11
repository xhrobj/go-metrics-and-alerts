.PHONY: \
	generate-reset \
	build build-server build-agent \
	clean-generated clean \
	test test-race test-coverage show-coverage \
	vet lint staticlint ci \
	postgres-up postgres-start postgres-stop postgres-rm postgres-connect \
	run-server run-server-env \
	run-agent run-agent-env

# параметры локального PostgreSQL-контейнера
POSTGRES_USER=metrics
POSTGRES_PASSWORD=password
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=metricsdb
POSTGRES_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# адреса Сервера для запуска через флаги и переменные окружения
SERVER_ADDRESS_DEFAULT=localhost:8080
SERVER_ADDRESS_ENV=localhost:8088

# параметры локального запуска Сервера и Агента
RATE_LIMIT=3
SECRET_KEY=god
AUDIT_FILE=audit.log

# пути для собранных бинарников Сервера и Агента
SERVER=cmd/server/server
AGENT=cmd/agent/agent

# запустить генератор reset.gen.go (см. С7И21)
generate-reset:
	go run ./cmd/reset

build: build-server build-agent

build-server:
	go build -o $(SERVER) ./cmd/server

build-agent:
	go build -o $(AGENT) ./cmd/agent

# удалить сгенерированные reset.gen.go, кроме фикстур
clean-generated:
	find . -name 'reset.gen.go' ! -path './cmd/reset/testdata/*' -exec rm -f {} +

# очистить артефакты сборки, coverage, тестовые бинарники и сгенерированные reset.gen.go
clean: clean-generated
	rm -f $(SERVER) $(AGENT) coverage.out
	find . -name "*.test" -delete

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...

show-coverage: test-coverage
	go tool cover -func=coverage.out | tail -n 1

vet:
	go vet ./...

lint:
	golangci-lint run ./...

# запустить multichecker с анализатором noosexit (см. С7И20)
staticlint:
	go run ./cmd/staticlint ./...

ci: build test-race vet lint staticlint

# создать и запустить новый Docker-контейнер PostgreSQL
postgres-up:
	docker run --name metrics-postgres \
		-e POSTGRES_USER=$(POSTGRES_USER) \
		-e POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
		-e POSTGRES_DB=$(POSTGRES_DB) \
		-p $(POSTGRES_PORT):5432 \
		-d postgres:16

# запустить уже созданный Docker-контейнер PostgreSQL
postgres-start:
	docker start metrics-postgres

# остановить Docker-контейнер PostgreSQL
postgres-stop:
	docker stop metrics-postgres

# удалить Docker-контейнер PostgreSQL
postgres-rm:
	docker rm -f metrics-postgres

# подключиться к PostgreSQL через psql внутри контейнера
postgres-connect:
	docker exec -it metrics-postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# собрать и запустить Сервер с параметрами командной строки
run-server: build-server
	./$(SERVER) -a=$(SERVER_ADDRESS_DEFAULT) -d=$(POSTGRES_DSN) -k=$(SECRET_KEY) --audit-file=$(AUDIT_FILE)

# собрать и запустить Сервер с параметрами через переменные окружения
run-server-env: build-server
	ADDRESS=$(SERVER_ADDRESS_ENV) DATABASE_DSN=$(POSTGRES_DSN) KEY=$(SECRET_KEY) AUDIT_FILE=$(AUDIT_FILE) ./$(SERVER)

# собрать и запустить Агент с параметрами командной строки
run-agent: build-agent
	./$(AGENT) -a=$(SERVER_ADDRESS_DEFAULT) -p=2 -r=10 -l=$(RATE_LIMIT) -k=$(SECRET_KEY)

# собрать и запустить Агент с параметрами через переменные окружения
run-agent-env: build-agent
	ADDRESS=$(SERVER_ADDRESS_ENV) POLL_INTERVAL=5 REPORT_INTERVAL=15 RATE_LIMIT=$(RATE_LIMIT) KEY=$(SECRET_KEY) ./$(AGENT)
