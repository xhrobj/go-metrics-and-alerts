# internal/encryption/testdata

Тестовые RSA-ключи для проверки пакета `internal/encryption`.

Ключи только для unit-тестов и не предназначены для запуска Агента и Сервера.

- `private.pem` - приватный ключ Сервера для тестовой расшифровки
- `public.pem` - публичный ключ Сервера, используемый Агентом для тестового шифрования

## Генерация этой пары ключей

1. Генерируем приватный ключ:

```bash
openssl genrsa \
  -out internal/encryption/testdata/private.pem \
  2048
```

2. Получаем публичный ключ из приватного:

```bash
openssl rsa \
  -in internal/encryption/testdata/private.pem \
  -pubout \
  -out internal/encryption/testdata/public.pem
```

## Проверка приватного ключа

```bash
openssl rsa \
  -in internal/encryption/testdata/private.pem \
  -check \
  -noout
```

Ожидаемый результат:

```text
RSA key ok
```

NOTE: мне кажется, на ключи GitHub и Sonar должны ругаться - сейчас проверим ...
