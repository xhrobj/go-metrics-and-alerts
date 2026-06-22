# internal/encryption

Пакет содержит загрузку RSA-ключей и гибридное шифрование сообщений с использованием RSA-OAEP и AES-GCM.

## Схема шифрования

Для каждого сообщения Агент:

1. генерирует одноразовый AES-ключ
2. шифрует данные через AES-GCM
3. шифрует AES-ключ публичным RSA-ключом через RSA-OAEP
4. объединяет зашифрованный ключ, nonce и ciphertext в транспортный payload

Сервер выполняет обратные операции приватным RSA-ключом.

Имя транспортного заголовка `Content-Encryption` объявлено в [`internal/protocol`](../protocol/README.md), а значение схемы `rsa-aes-gcm` остаётся в пакете `internal/encryption`.

## Ключи для тестов

Пакет `internal/encryption/testkeys` при каждом тесте:

- генерирует временную RSA-пару
- записывает публичный ключ в формате PKIX
- записывает приватный ключ в формате PKCS#8
- сохраняет PEM-файлы в `t.TempDir()`

Временные файлы автоматически удаляются после теста.

## Ключи для локального запуска

```bash
make crypto-keys
```

Ключи сохраняются в игнорируемом Git каталоге:

```text
.keys/
├── private.pem
└── public.pem
```

- `private.pem` используется Сервером
- `public.pem` используется Агентом

Запуск с шифрованием:

```bash
make run-server-crypto
make run-agent-crypto
```

## Ручная генерация ключей

Приватный ключ:

```bash
openssl genrsa   -out .keys/private.pem   2048
```

Публичный ключ:

```bash
openssl rsa   -in .keys/private.pem   -pubout   -out .keys/public.pem
```

Проверка соответствия пары:

```bash
openssl pkey   -in .keys/private.pem   -pubout   -outform DER |
openssl sha256

openssl pkey   -pubin   -in .keys/public.pem   -outform DER |
openssl sha256
```

Обе команды должны вывести одинаковый SHA-256.
