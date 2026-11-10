# Swiss

[English](README.md) · [Русский](README.ru.md)

Swiss is a Telegram bot with a pocket set of tools in Go: URL shortener (with an HTTP API), IP and domain lookup with a map pin and WHOIS. A single binary, no database. More tools are coming.

## Features

- Shorten links through the HTTP API (`POST /shorten`) or the Telegram bot
- Short links redirect with `301`
- 6-character IDs from a cryptographically secure random source (`crypto/rand`)
- Only users on the allow-list can shorten links in the bot
- IP and domain lookup in the bot (`/ip`, `/info`) and in inline mode, open to everyone: map pin, country with flag, region, city, ZIP, timezone with local time, ISP, organization, AS, hostname, mobile flag, domain WHOIS (registrar, dates, expiry warning), English and Russian
- Dependencies: [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api), `golang.org/x/net` (IDN)

## Getting Started

Requires Go and a bot token from [@BotFather](https://t.me/BotFather).

```bash
git clone https://github.com/CraftStick/LinkShrinker.git
cd LinkShrinker
BOT_TOKEN=123456:ABC... go run .
```

The server listens on `:8080`.

## Configuration

| Setting | Where | Description |
|---|---|---|
| `BOT_TOKEN` | environment variable | Telegram bot token |
| `domain` | `main.go` | Public base URL used in short links |
| `allowedUsers` | `main.go` | Telegram user IDs allowed to use the bot |
| `port` | `main.go` | HTTP port, `:8080` by default |

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

A URL without `http://` or `https://` gets `http://` added.

**`GET /{id}`** redirects to the original URL, or returns `404` if the ID is unknown.

## Telegram bot

Send `/start`, then send any link, and the bot replies with the short one (allow-listed users only).

Lookup is open to everyone:

- `/ip <address>` or `/info <address>`: IP, domain or link, IDN domains included
- Inline: type `@your_bot 8.8.8.8` in any chat and pick the card or the map pin
- Geolocation: [ip-api.com](https://ip-api.com) (45 requests/minute, quota tracked from its headers; 10 are kept for chat messages), with [ipwho.is](https://ipwho.is) as a backup
- Domains: RDAP (IANA bootstrap) with a WHOIS port 43 fallback, so `.ru`, `.рф` and other ccTLDs work
- Results are cached (5 minutes for geolocation, 1 hour for WHOIS)

Based on [IPBot-telegram](https://github.com/CraftStick/IPBot-telegram).

## Test

```bash
go test ./...
```

## Build

```bash
go build -o shortener .
GOOS=linux GOARCH=amd64 go build -o shortener-linux .
```

## Limitations

- Links live in memory and are lost on restart
- No rate limiting on `/shorten`
