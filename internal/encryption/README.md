# internal/encryption

Пакет содержит загрузку RSA-ключей и гибридное шифрование сообщений с использованием RSA-OAEP и AES-GCM.

## Ключи для тестов

Пакет `internal/encryption/testkeys` при каждом запуске теста:

- генерирует временную RSA-пару
- записывает публичный ключ в формате PKIX
- записывает приватный ключ в формате PKCS#8
- сохраняет PEM-файлы в `t.TempDir()`

Временные файлы автоматически удаляются после завершения теста.

## Ключи для локального запуска

Локальную RSA-пару для запуска Агента и Сервера можно создать командой:

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

## Ручная генерация этой пары ключей

Ну и напоминалка как сгенерить ключи руками (не из `Makefile`).

1. Генерируем приватный RSA-ключ:

```bash
openssl genrsa \
  -out .keys/private.pem \
  2048
```

2. Получаем публичный ключ из приватного:

```bash
openssl rsa \
  -in .keys/private.pem \
  -pubout \
  -out .keys/public.pem
```

3. Проверка, что публичный ключ соответствует приватному:

```bash
openssl pkey \
  -in .keys/private.pem \
  -pubout \
  -outform DER |
openssl sha256

openssl pkey \
  -pubin \
  -in .keys/public.pem \
  -outform DER |
openssl sha256
```

Обе команды должны вывести одинаковый SHA-256.
