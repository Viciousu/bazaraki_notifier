# Bazaraki Notifier

Telegram bot that monitors [bazaraki.com](https://www.bazaraki.com) listing pages and sends notifications when new advertisements appear.

## Telegram Bot Setup

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` and follow the prompts:
   - Choose a display name (e.g. "Bazaraki Notifier")
   - Choose a username (must end with `bot`, e.g. `my_bazaraki_bot`)
3. BotFather will reply with your **bot token** — a string like `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`. Save it.
4. (Optional) To receive notifications about new users, get your chat ID:
   - Send any message to your bot
   - Open `https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates`
   - Find `"chat":{"id": 123456789}` — that's your chat ID for `NOTIFY_TO_CHAT`

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `TOKEN` | Yes | — | Telegram bot token from BotFather |
| `DATA_FOLDER` | Yes | — | Path to store subscription data |
| `CHECKING_INTERVAL` | No | `300` | Interval between checks in seconds |
| `NOTIFY_TO_CHAT` | No | — | Chat ID to notify about new users |
| `BATCH_SIZE` | No | `20` | Max number of ads per Telegram message |
| `USER_AGENT` | No | Chrome UA | Custom User-Agent header for HTTP requests |

## Usage

Send your bot a bazaraki.com listing URL, for example:

```
https://www.bazaraki.com/real-estate/houses-and-villas-rent/lemesos-district-limassol/?price_min=500&price_max=1000
```

The bot will check the page periodically and send you new listings as they appear.

Commands:
- `/start` — introduction message
- `/stop` — unsubscribe from all notifications

## Run with Docker

### Build

For the current platform:

```bash
docker build -t bazaraki-notifier .
```

Cross-compile for a remote amd64 server (e.g. from Apple Silicon Mac):

```bash
docker build --platform linux/amd64 -t bazaraki-notifier:amd64 .
```

### Run

`--network host` is recommended — avoids Docker NAT which can affect Cloudflare bot detection:

```bash
docker run -d \
  --name bazaraki-notifier \
  --network host \
  --restart unless-stopped \
  -e TOKEN=your_telegram_bot_token \
  -e DATA_FOLDER=/app/data \
  -e CHECKING_INTERVAL=600 \
  -v bazaraki-data:/app/data \
  bazaraki-notifier
```

With all optional settings:

```bash
docker run -d \
  --name bazaraki-notifier \
  --network host \
  --restart unless-stopped \
  -e TOKEN=your_telegram_bot_token \
  -e DATA_FOLDER=/app/data \
  -e CHECKING_INTERVAL=600 \
  -e NOTIFY_TO_CHAT=your_chat_id \
  -e BATCH_SIZE=20 \
  -v bazaraki-data:/app/data \
  bazaraki-notifier
```

### Deploy to a remote server

```bash
docker save bazaraki-notifier:amd64 | ssh user@server "docker load"
```

### View logs

```bash
docker logs -f bazaraki-notifier
```

### Update

```bash
docker stop bazaraki-notifier && docker rm bazaraki-notifier
# Then run again with the new image
```

### Persistent data

Subscription data is stored in the `bazaraki-data` Docker volume. It survives container restarts, stops, and re-creation. To reset all subscriptions:

```bash
docker volume rm bazaraki-data
```

## Run without Docker

```bash
go build
DATA_FOLDER=./data TOKEN=your_telegram_bot_token CHECKING_INTERVAL=300 ./bazaraki_notifier
```

## Run Tests

Integration tests make real HTTP requests to bazaraki.com:

```bash
go test -tags=integration -v
```
