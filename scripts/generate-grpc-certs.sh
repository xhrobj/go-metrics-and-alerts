#!/usr/bin/env bash

set -euo pipefail

CERT_DIR=".certs"
CERT_DAYS=3650

CA_CERT="${CERT_DIR}/grpc-ca.pem"
CA_KEY="${CERT_DIR}/grpc-ca-key.pem"

SERVER_CERT="${CERT_DIR}/grpc-server.pem"
SERVER_KEY="${CERT_DIR}/grpc-server-key.pem"

SERVER_CSR="${CERT_DIR}/grpc-server.csr"
SERVER_EXT="${CERT_DIR}/grpc-server.ext"

# NOTE: проверяем только наличие файлов, а не их корректность
if [[ -f "${CA_CERT}" &&
	-f "${CA_KEY}" &&
	-f "${SERVER_CERT}" &&
	-f "${SERVER_KEY}" ]]; then
	echo "(^_^) gRPC TLS certificates already exist"
	exit 0
fi

mkdir -p "${CERT_DIR}"

rm -f \
	"${CA_CERT}" \
	"${CA_KEY}" \
	"${SERVER_CERT}" \
	"${SERVER_KEY}" \
	"${SERVER_CSR}" \
	"${SERVER_EXT}"

cat >"${SERVER_EXT}" <<'EOF'
[server_cert]
basicConstraints = critical, CA:FALSE
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = DNS:localhost,DNS:server,IP:127.0.0.1,IP:::1
EOF

umask 077

# создать ключ и самоподписанный сертификат центра сертификации
openssl genrsa -out "${CA_KEY}" 2048

openssl req \
	-x509 \
	-new \
	-sha256 \
	-key "${CA_KEY}" \
	-days "${CERT_DAYS}" \
	-subj "/CN=go-metrics-and-alerts gRPC CA" \
	-out "${CA_CERT}"

# создать ключ и запрос на сертификат gRPC-сервера
openssl genrsa -out "${SERVER_KEY}" 2048

openssl req \
	-new \
	-sha256 \
	-key "${SERVER_KEY}" \
	-subj "/CN=server" \
	-out "${SERVER_CSR}"

# подписать сертификат сервера созданным центром сертификации
openssl x509 \
	-req \
	-sha256 \
	-in "${SERVER_CSR}" \
	-CA "${CA_CERT}" \
	-CAkey "${CA_KEY}" \
	-set_serial 1 \
	-days "${CERT_DAYS}" \
	-extfile "${SERVER_EXT}" \
	-extensions server_cert \
	-out "${SERVER_CERT}"

rm -f "${SERVER_CSR}" "${SERVER_EXT}"

chmod 600 "${CA_KEY}" "${SERVER_KEY}"
chmod 644 "${CA_CERT}" "${SERVER_CERT}"

openssl verify \
	-CAfile "${CA_CERT}" \
	"${SERVER_CERT}"

echo "(*_*) gRPC TLS certificates generated in ${CERT_DIR}"
