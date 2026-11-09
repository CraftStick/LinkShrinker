# LinkShrinker

Minimal URL shortener in Go: an HTTP API plus a Telegram bot, in a single binary with no database.

## Features

- Shortens links through the **HTTP API** (`POST /shorten`) or the **Telegram bot**
- Redirects short links with `301`
- 6-character IDs from a cryptographically secure random source (`crypto/rand`)
- The bot answers only users from an allow-list
- One dependency: [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)

## Quick start

```bash
git clone https://github.com/CraftStick/LinkShrinker.git
cd LinkShrinker
BOT_TOKEN=123456:ABC... go run .
```

The server listens on `:8080`.

## Configuration

| Setting | Where | Description |
|---|---|---|
| `BOT_TOKEN` | environment variable | Telegram bot token from [@BotFather](https://t.me/BotFather) |
| `domain` | `main.go` | Public base URL used in short links |
| `allowedUsers` | `main.go` | Telegram user IDs allowed to use the bot |
| `port` | `main.go` | HTTP port (default `:8080`) |

## API

### `POST /shorten`

```bash
curl -X POST http://localhost:8080/shorten \
  -H 'Content-Type: application/json' \
  -d '{"url": "example.com/very/long/path"}'
```

```json
{"short_url": "https://your-domain/aB3xY9"}
```

URLs without `http://` or `https://` get `http://` prepended.

### `GET /{id}`

Redirects to the original URL, or returns `404` if the ID is unknown.

## Telegram bot

Send `/start`, then send any link and the bot replies with the short one.

## Build

```bash
go build -o shortener .
GOOS=linux GOARCH=amd64 go build -o shortener-linux .
```

## Limitations

- Links are kept **in memory** and disappear on restart
- No rate limiting on `/shorten`
