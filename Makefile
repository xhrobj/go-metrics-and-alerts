.PHONY: \
	show-coverage \
	generate-reset generate-mocks generate-proto \
	crypto-keys \
	build build-server build-agent \
	clean-generated clean \
	test test-race test-coverage \
	vet lint staticlint ci \
	postgres-up postgres-start postgres-stop postgres-rm postgres-connect \
	run-server run-server-env run-server-config run-server-crypto \
	run-agent run-agent-env run-agent-config run-agent-crypto \
	compose-up compose-down compose-logs

# локальные параметры из env-файла
ENV_FILE ?= .env
-include $(ENV_FILE)

# параметры локального PostgreSQL-контейнера
POSTGRES_USER ?= metrics
POSTGRES_PASSWORD ?= password
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB ?= metricsdb
POSTGRES_DSN=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# адреса HTTP- и gRPC-Сервера для локального запуска
SERVER_HTTP_ADDRESS ?= localhost:8080
SERVER_HTTP_ADDRESS_ENV ?= localhost:8088
SERVER_GRPC_ADDRESS ?= localhost:50051
SERVER_GRPC_ADDRESS_ENV ?= localhost:50058

# параметры локального запуска Сервера и Агента
RATE_LIMIT ?= 3
SECRET_KEY ?= god
TRUSTED_SUBNET ?=
AUDIT_FILE ?= audit.log

# пути для собранных бинарников Сервера и Агента
SERVER=cmd/server/server
AGENT=cmd/agent/agent

# номер текущего спринта (участвует в формировании build version)
SPRINT_NUMBER = 9

# данные о сборке подставляются в бинарники Агента и Сервера через ldflags (см. С7И23)
BUILD_VERSION = v0.$(SPRINT_NUMBER).0
BUILD_DATE = $(shell date +%Y-%m-%d)
BUILD_COMMIT = $(shell git rev-parse --short HEAD)

LDFLAGS = -X main.buildVersion=$(BUILD_VERSION) -X main.buildDate=$(BUILD_DATE) -X main.buildCommit=$(BUILD_COMMIT)

# локальная пара RSA-ключей
CRYPTO_DIR=.keys
SERVER_PRIVATE_KEY=$(CRYPTO_DIR)/private.pem
AGENT_PUBLIC_KEY=$(CRYPTO_DIR)/public.pem

# пути к example-конфигам Сервера и Агента
CONFIGS_DIR := configs
SERVER_CONFIG := $(CONFIGS_DIR)/server.example.json
AGENT_CONFIG := $(CONFIGS_DIR)/agent.example.json

# параметры генерации protobuf-кода
GO_MODULE := github.com/xhrobj/go-metrics-and-alerts
PROTO_FILE := api/metrics.proto

show-coverage: test-coverage
	go tool cover -func=coverage.out | tail -n 1

# запустить генератор reset.gen.go (см. С7И21)
generate-reset:
	go run ./cmd/reset

# сгенерировать моки пакета handler
generate-mocks:
	go generate ./internal/server/transport/http/handler

# сгенерировать Go-код protobuf-сообщений и gRPC-сервиса
generate-proto:
	protoc \
		--proto_path=. \
		--go_out=. \
		--go_opt=module=$(GO_MODULE) \
		--go_opt=default_api_level=API_OPAQUE \
		--go-grpc_out=. \
		--go-grpc_opt=module=$(GO_MODULE) \
		$(PROTO_FILE)

# создать (при необходимости) локальную RSA-пару
crypto-keys:
	@if [ ! -f "$(SERVER_PRIVATE_KEY)" ] || [ ! -f "$(AGENT_PUBLIC_KEY)" ]; then \
		mkdir -p "$(CRYPTO_DIR)"; \
		openssl genrsa -out "$(SERVER_PRIVATE_KEY)" 2048; \
		openssl rsa \
			-in "$(SERVER_PRIVATE_KEY)" \
			-pubout \
			-out "$(AGENT_PUBLIC_KEY)"; \
	fi

build: build-server build-agent

build-server:
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(SERVER) \
		./cmd/server

build-agent:
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(AGENT) \
		./cmd/agent

# удалить сгенерированный Go-код, кроме тестовых фикстур reset и тестовых моков
clean-generated:
	find . -name 'reset.gen.go' ! -path './cmd/reset/testdata/*' -exec rm -f {} +
	rm -f internal/proto/*.pb.go

# очистить артефакты сборки, coverage и тестовые бинарники
clean:
	rm -f $(SERVER) $(AGENT) coverage.out
	find . -name "*.test" -delete

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...

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
	./$(SERVER) \
		-a=$(SERVER_HTTP_ADDRESS) \
		-g=$(SERVER_GRPC_ADDRESS) \
		-d=$(POSTGRES_DSN) \
		-k=$(SECRET_KEY) \
		-t=$(TRUSTED_SUBNET) \
		--audit-file=$(AUDIT_FILE)

# собрать и запустить Сервер с параметрами через переменные окружения
run-server-env: build-server
	ADDRESS=$(SERVER_HTTP_ADDRESS_ENV) \
	GRPC_ADDRESS=$(SERVER_GRPC_ADDRESS_ENV) \
	DATABASE_DSN=$(POSTGRES_DSN) \
	KEY=$(SECRET_KEY) \
	TRUSTED_SUBNET=$(TRUSTED_SUBNET) \
	AUDIT_FILE=$(AUDIT_FILE) \
	./$(SERVER)

# собрать и запустить Сервер с параметрами из JSON-файла
run-server-config: build-server crypto-keys
	./$(SERVER) --config $(SERVER_CONFIG)

# собрать и запустить Сервер с приватным ключом
run-server-crypto: build-server crypto-keys
	./$(SERVER) \
		-a=$(SERVER_HTTP_ADDRESS) \
		-g=$(SERVER_GRPC_ADDRESS) \
		-k=$(SECRET_KEY) \
		--crypto-key=$(SERVER_PRIVATE_KEY) \
		-t=$(TRUSTED_SUBNET)

# собрать и запустить Агент с параметрами командной строки
run-agent: build-agent
	./$(AGENT) -a=$(SERVER_HTTP_ADDRESS) -p=2 -r=10 -l=$(RATE_LIMIT) -k=$(SECRET_KEY)

# собрать и запустить Агент с параметрами через переменные окружения
run-agent-env: build-agent
	ADDRESS=$(SERVER_HTTP_ADDRESS_ENV) POLL_INTERVAL=5 REPORT_INTERVAL=15 RATE_LIMIT=$(RATE_LIMIT) KEY=$(SECRET_KEY) ./$(AGENT)

# собрать и запустить Агент с параметрами из JSON-файла
run-agent-config: build-agent crypto-keys
	./$(AGENT) --config $(AGENT_CONFIG)

# собрать и запустить Агент с публичным ключом
run-agent-crypto: build-agent crypto-keys
	./$(AGENT) \
		-a=$(SERVER_HTTP_ADDRESS) \
		-p=2 \
		-r=5 \
		-l=$(RATE_LIMIT) \
		-k=$(SECRET_KEY) \
		--crypto-key=$(AGENT_PUBLIC_KEY)

# собрать и запустить PostgreSQL, Сервер и Агент через Docker Compose
compose-up: crypto-keys
	BUILD_VERSION=$(BUILD_VERSION) \
	BUILD_DATE=$(BUILD_DATE) \
	BUILD_COMMIT=$(BUILD_COMMIT) \
	POSTGRES_USER=$(POSTGRES_USER) \
	POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	POSTGRES_DB=$(POSTGRES_DB) \
	RATE_LIMIT=$(RATE_LIMIT) \
	SECRET_KEY=$(SECRET_KEY) \
	docker compose up --build -d

# остановить и удалить контейнеры Docker Compose
compose-down:
	docker compose down

# показать логи сервисов Docker Compose
compose-logs:
	docker compose logs -f
