FROM golang:1.26-alpine AS builder

ARG BUILD_VERSION
ARG BUILD_DATE
ARG BUILD_COMMIT

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/server ./cmd/server
COPY cmd/agent ./cmd/agent
COPY internal ./internal

RUN mkdir -p /out

FROM builder AS server-builder

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${BUILD_COMMIT}" \
	-o /out/server \
	./cmd/server

FROM builder AS agent-builder

RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags "-X main.buildVersion=${BUILD_VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${BUILD_COMMIT}" \
	-o /out/agent \
	./cmd/agent

FROM alpine:3.23 AS server

WORKDIR /app

RUN apk add --no-cache curl \
	&& adduser -D -g '' appuser

COPY --from=server-builder /out/server /app/server
COPY migrations /app/migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]

FROM alpine:3.23 AS agent

WORKDIR /app

RUN adduser -D -g '' appuser

COPY --from=agent-builder /out/agent /app/agent

USER appuser

ENTRYPOINT ["/app/agent"]
