# LinkShrinker

[English](README.md) · [Русский](README.ru.md)

Минимальный сокращатель ссылок на Go: HTTP API и Telegram-бот в одном бинарнике, без базы данных.

## Возможности

- Сокращает ссылки через HTTP API (`POST /shorten`) или Telegram-бота
- Короткие ссылки перенаправляют кодом `301`
- 6-символьные ID из криптостойкого источника случайности (`crypto/rand`)
- Бот отвечает только пользователям из белого списка
- Одна зависимость: [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)

## Запуск

Нужны Go и токен бота от [@BotFather](https://t.me/BotFather).

```bash
git clone https://github.com/CraftStick/LinkShrinker.git
cd LinkShrinker
BOT_TOKEN=123456:ABC... go run .
```

Сервер слушает порт `:8080`.

## Настройки

| Настройка | Где | Описание |
|---|---|---|
| `BOT_TOKEN` | переменная окружения | Токен Telegram-бота |
| `domain` | `main.go` | Публичный адрес для коротких ссылок |
| `allowedUsers` | `main.go` | ID пользователей Telegram, которым разрешён бот |
| `port` | `main.go` | HTTP-порт, по умолчанию `:8080` |

## API

**`POST /shorten`**

```bash
curl -X POST http://localhost:8080/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url": "example.com/very/long/path"}'
```

```json
{"short_url": "https://your-domain/aB3xY9"}
```

Если в ссылке нет `http://` или `https://`, добавляется `http://`.

**`GET /{id}`** перенаправляет на исходную ссылку или возвращает `404`, если ID неизвестен.

## Telegram-бот

Отправьте `/start`, затем любую ссылку: бот ответит короткой.

## Сборка

```bash
go build -o shortener .
GOOS=linux GOARCH=amd64 go build -o shortener-linux .
```

## Ограничения

- Ссылки хранятся в памяти и пропадают при перезапуске
- У `/shorten` нет ограничения частоты запросов
