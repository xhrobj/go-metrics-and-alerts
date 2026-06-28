.PHONY: \
	show-coverage \
	generate-reset generate-mocks generate-proto \
	crypto-keys grpc-certs \
	build build-server build-agent \
	clean-generated clean \
	test test-race test-coverage \
	vet lint staticlint ci \
	postgres-up postgres-start postgres-stop postgres-rm postgres-connect \
	run-server-debug run-server run-server-env run-server-config run-server-crypto \
	run-agent run-agent-env run-agent-config run-agent-crypto \
	compose-up compose-down compose-logs

# локальные параметры из env-файла
ENV_FILE ?= .env
-include $(ENV_FILE)

# пути к example-конфигам Сервера и Агента
# !!!: для проверки JSON-конфигурации укажите свои файлы с недефолтными значениями
CONFIGS_DIR ?= configs
SERVER_CONFIG ?= $(CONFIGS_DIR)/server.example.json
AGENT_CONFIG ?= $(CONFIGS_DIR)/agent.example.json

# параметры генерации protobuf-кода
GO_MODULE := github.com/xhrobj/go-metrics-and-alerts
PROTO_FILE := api/metrics.proto

# локальная RSA-пара для шифрования HTTP-запросов
HTTP_CRYPTO_DIR := .keys
HTTP_CRYPTO_PRIVATE_KEY := $(HTTP_CRYPTO_DIR)/http-crypto-private.pem
HTTP_CRYPTO_PUBLIC_KEY := $(HTTP_CRYPTO_DIR)/http-crypto-public.pem

# сертификаты и ключи gRPC TLS
GRPC_CERT_DIR := .certs
GRPC_CERT_SCRIPT := scripts/generate-grpc-certs.sh
GRPC_CA_CERT := $(GRPC_CERT_DIR)/grpc-ca.pem
GRPC_SERVER_CERT := $(GRPC_CERT_DIR)/grpc-server.pem
GRPC_SERVER_KEY := $(GRPC_CERT_DIR)/grpc-server-key.pem

# пути для собранных бинарников Сервера и Агента
SERVER := cmd/server/server
AGENT := cmd/agent/agent

# номер текущего спринта (участвует в формировании build version)
SPRINT_NUMBER := 9

# данные о сборке подставляются в бинарники Агента и Сервера через ldflags (см. С7И23)
BUILD_VERSION := v0.$(SPRINT_NUMBER).0
BUILD_DATE := $(shell date +%Y-%m-%d)
BUILD_COMMIT := $(shell git rev-parse --short HEAD)

LDFLAGS := -X main.buildVersion=$(BUILD_VERSION) -X main.buildDate=$(BUILD_DATE) -X main.buildCommit=$(BUILD_COMMIT)

# параметры локального PostgreSQL-контейнера
POSTGRES_USER ?= metrics
POSTGRES_PASSWORD ?= password
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5432
POSTGRES_DB ?= metricsdb
POSTGRES_DSN ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# HTTP/gRPC-адреса Сервера для локального запуска
SERVER_HTTP_ADDRESS ?= localhost:8080
SERVER_HTTP_ADDRESS_ENV ?= localhost:8088
SERVER_GRPC_ADDRESS ?= localhost:50051
SERVER_GRPC_ADDRESS_ENV ?= localhost:50058

# транспорт и адрес Агента для локального запуска
AGENT_TRANSPORT ?= grpc
AGENT_SERVER_ADDRESS ?= $(if $(filter http,$(AGENT_TRANSPORT)),$(SERVER_HTTP_ADDRESS),$(SERVER_GRPC_ADDRESS))
AGENT_SERVER_ADDRESS_ENV ?= $(if $(filter http,$(AGENT_TRANSPORT)),$(SERVER_HTTP_ADDRESS_ENV),$(SERVER_GRPC_ADDRESS_ENV))
COMPOSE_AGENT_ADDRESS ?= $(if $(filter http,$(AGENT_TRANSPORT)),server:8080,server:50051)

# параметры локального запуска Сервера и Агента
RATE_LIMIT ?= 3
SECRET_KEY ?= god
TRUSTED_SUBNET ?=
AUDIT_FILE ?= audit.log

# обновить профиль покрытия и вывести общий процент
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

# создать (при необходимости) локальную RSA-пару для шифрования HTTP-запросов
crypto-keys:
	@if [ ! -f "$(HTTP_CRYPTO_PRIVATE_KEY)" ] || [ ! -f "$(HTTP_CRYPTO_PUBLIC_KEY)" ]; then \
		mkdir -p "$(HTTP_CRYPTO_DIR)"; \
		openssl genrsa -out "$(HTTP_CRYPTO_PRIVATE_KEY)" 2048; \
		openssl rsa \
			-in "$(HTTP_CRYPTO_PRIVATE_KEY)" \
			-pubout \
			-out "$(HTTP_CRYPTO_PUBLIC_KEY)"; \
	fi

# создать (при необходимости) сертификаты и ключи gRPC TLS
grpc-certs:
	./$(GRPC_CERT_SCRIPT)

# собрать бинарники Сервера и Агента
build: build-server build-agent

# собрать бинарник Сервера с build info
build-server:
	go build \
		-ldflags "$(LDFLAGS)" \
		-o $(SERVER) \
		./cmd/server

# собрать бинарник Агента с build info
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

# запустить все тесты
test:
	go test ./...

# запустить все тесты с детектором гонок данных
test-race:
	go test -race ./...

# запустить тесты и сохранить атомарный профиль покрытия
test-coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...

# выполнить стандартный статический анализ Go-кода
vet:
	go vet ./...

# проверить проект набором линтеров golangci-lint
lint:
	golangci-lint run ./...

# запустить multichecker с анализатором noosexit (см. С7И20)
staticlint:
	go run ./cmd/staticlint ./...

# собрать проект и выполнить полный набор CI-проверок
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

# собрать и запустить Сервер с debug-логированием
run-server-debug:
	LOG_LEVEL=debug $(MAKE) run-server

# собрать и запустить Сервер с параметрами командной строки
run-server: build-server grpc-certs
	./$(SERVER) \
		-a=$(SERVER_HTTP_ADDRESS) \
		-g=$(SERVER_GRPC_ADDRESS) \
		--grpc-tls-cert=$(GRPC_SERVER_CERT) \
		--grpc-tls-key=$(GRPC_SERVER_KEY) \
		-d=$(POSTGRES_DSN) \
		-k=$(SECRET_KEY) \
		-t=$(TRUSTED_SUBNET) \
		--audit-file=$(AUDIT_FILE)

# собрать и запустить Сервер с параметрами через переменные окружения
run-server-env: build-server grpc-certs
	ADDRESS=$(SERVER_HTTP_ADDRESS_ENV) \
	GRPC_ADDRESS=$(SERVER_GRPC_ADDRESS_ENV) \
	GRPC_TLS_CERT=$(GRPC_SERVER_CERT) \
	GRPC_TLS_KEY=$(GRPC_SERVER_KEY) \
	DATABASE_DSN=$(POSTGRES_DSN) \
	KEY=$(SECRET_KEY) \
	TRUSTED_SUBNET=$(TRUSTED_SUBNET) \
	AUDIT_FILE=$(AUDIT_FILE) \
	./$(SERVER)

# собрать и запустить Сервер с параметрами из JSON-файла
run-server-config: build-server crypto-keys grpc-certs
	./$(SERVER) --config $(SERVER_CONFIG)

# собрать и запустить Сервер с RSA-расшифровкой HTTP-запросов и gRPC TLS
run-server-crypto: build-server crypto-keys grpc-certs
	./$(SERVER) \
		-a=$(SERVER_HTTP_ADDRESS) \
		-g=$(SERVER_GRPC_ADDRESS) \
		--grpc-tls-cert=$(GRPC_SERVER_CERT) \
		--grpc-tls-key=$(GRPC_SERVER_KEY) \
		-k=$(SECRET_KEY) \
		--crypto-key=$(HTTP_CRYPTO_PRIVATE_KEY) \
		-t=$(TRUSTED_SUBNET)

# собрать и запустить Агент с параметрами командной строки
run-agent: build-agent grpc-certs
	./$(AGENT) \
		-a=$(AGENT_SERVER_ADDRESS) \
		--transport=$(AGENT_TRANSPORT) \
		--grpc-tls-ca=$(GRPC_CA_CERT) \
		-p=2 \
		-r=10 \
		-l=$(RATE_LIMIT) \
		-k=$(SECRET_KEY)

# собрать и запустить Агент с параметрами через переменные окружения
run-agent-env: build-agent grpc-certs
	ADDRESS=$(AGENT_SERVER_ADDRESS_ENV) \
	TRANSPORT=$(AGENT_TRANSPORT) \
	GRPC_TLS_CA=$(GRPC_CA_CERT) \
	POLL_INTERVAL=5 \
	REPORT_INTERVAL=15 \
	RATE_LIMIT=$(RATE_LIMIT) \
	KEY=$(SECRET_KEY) \
	./$(AGENT)

# собрать и запустить Агент с параметрами из JSON-файла
run-agent-config: build-agent crypto-keys grpc-certs
	./$(AGENT) --config $(AGENT_CONFIG)

# собрать и запустить HTTP-Агент с RSA-шифрованием тела запросов
run-agent-crypto: build-agent crypto-keys
	./$(AGENT) \
		-a=$(SERVER_HTTP_ADDRESS) \
		--transport=http \
		-p=2 \
		-r=5 \
		-l=$(RATE_LIMIT) \
		-k=$(SECRET_KEY) \
		--crypto-key=$(HTTP_CRYPTO_PUBLIC_KEY)

# собрать и запустить PostgreSQL, Сервер и Агент через Docker Compose
compose-up: crypto-keys grpc-certs
	BUILD_VERSION=$(BUILD_VERSION) \
	BUILD_DATE=$(BUILD_DATE) \
	BUILD_COMMIT=$(BUILD_COMMIT) \
	POSTGRES_USER=$(POSTGRES_USER) \
	POSTGRES_PASSWORD=$(POSTGRES_PASSWORD) \
	POSTGRES_DB=$(POSTGRES_DB) \
	RATE_LIMIT=$(RATE_LIMIT) \
	SECRET_KEY=$(SECRET_KEY) \
	AGENT_TRANSPORT=$(AGENT_TRANSPORT) \
	AGENT_ADDRESS=$(COMPOSE_AGENT_ADDRESS) \
	docker compose up --build -d

# остановить и удалить контейнеры Docker Compose
compose-down:
	docker compose down

# показать логи сервисов Docker Compose
compose-logs:
	docker compose logs -f
