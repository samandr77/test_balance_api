# test_balance_api

REST API для вывода средств с баланса пользователя.

## Запуск

```bash
docker compose up --build -d
```

API доступно на `http://localhost:8080`.

## Аутентификация

Все запросы требуют заголовок:

```
Authorization: Bearer dev-token
```

## API

### Создать вывод средств

```
POST /v1/withdrawals
```

```bash
curl -s -X POST http://localhost:8080/v1/withdrawals \
  -H "Authorization: Bearer dev-token" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "00000000-0000-0000-0000-000000000001",
    "amount": "50",
    "currency": "USDT",
    "destination": "0xABC123",
    "idempotency_key": "unique-key-1"
  }'
```

Повторный запрос с тем же `idempotency_key` и телом вернёт `200` и существующую запись вместо создания новой.

### Получить вывод по ID

```
GET /v1/withdrawals/{id}
```

```bash
curl -s http://localhost:8080/v1/withdrawals/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx \
  -H "Authorization: Bearer dev-token"
```

## HTTP коды ответов

| Код | Причина |
|-----|---------|
| 201 | Вывод создан |
| 200 | Идемпотентный повтор |
| 400 | Невалидные данные (некорректный UUID, нулевая сумма и т.д.) |
| 401 | Неверный токен |
| 404 | Вывод не найден |
| 409 | Недостаточно средств |
| 422 | Баланс не инициализирован / конфликт idempotency key |
| 500 | Внутренняя ошибка |

## Тесты

```bash
# Unit-тесты сервиса
go test ./internal/service/...

# Интеграционные тесты репозитория (требует Docker)
go test -race ./internal/repository/...
```
