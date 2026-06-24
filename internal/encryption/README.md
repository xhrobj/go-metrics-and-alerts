# internal/encryption

Пакет загружает RSA-ключи из PEM-файлов и реализует гибридное шифрование сообщений с использованием RSA-OAEP и AES-GCM. Шифрование применяется при передаче метрик по HTTP.

## Схема шифрования

Для каждого сообщения Агент:

1. генерирует случайный 256-битный AES-ключ
2. создаёт случайный nonce необходимого для AES-GCM размера
3. шифрует исходные данные с помощью AES-GCM
4. шифрует AES-ключ публичным RSA-ключом через RSA-OAEP с SHA-256
5. сериализует результат в JSON-конверт с полями `key`, `nonce` и `data`

Сервер выполняет обратные операции: расшифровывает AES-ключ приватным RSA-ключом, а затем расшифровывает данные с помощью AES-GCM.

HTTP-транспорт обозначает зашифрованное тело заголовком `Content-Encryption` со значением `rsa-aes-gcm`. Имя заголовка объявлено в пакете `internal/protocol`, а значение схемы находится в пакете `internal/encryption`.

## Форматы ключей

Для публичных RSA-ключей поддерживаются форматы:

- PKIX
- PKCS#1

Для приватных RSA-ключей поддерживаются форматы:

- PKCS#8
- PKCS#1

## Ключи для тестов

Функция `testkeys.Generate(t)` из пакета `internal/encryption/testkeys` при каждом вызове:

- генерирует временную RSA-пару
- записывает публичный ключ в формате PKIX
- записывает приватный ключ в формате PKCS#8
- сохраняет PEM-файлы в каталоге, созданном через `t.TempDir()`

Временные файлы автоматически удаляются после завершения теста.

## Ключи для локального запуска

```bash
make crypto-keys
```

Команда при необходимости создаёт RSA-пару в каталоге `.keys/`, игнорируемом Git:

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

Создание каталога:

```bash
mkdir -p .keys
```

Приватный ключ:

```bash
openssl genrsa -out .keys/private.pem 2048
```

Публичный ключ:

```bash
openssl rsa -in .keys/private.pem -pubout -out .keys/public.pem
```

Проверка соответствия пары:

```bash
openssl pkey -in .keys/private.pem -pubout -outform DER | openssl sha256
openssl pkey -pubin -in .keys/public.pem -outform DER | openssl sha256
```

Обе команды должны вывести одинаковый SHA-256.
