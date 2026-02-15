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

## Run with Docker Compose (recommended)

1. Copy the example env file and fill in your token:

```bash
cp .env.example .env
# Edit .env and set TOKEN=your_telegram_bot_token
```

2. Start the bot:

```bash
docker compose up -d
```

3. View logs:

```bash
docker compose logs -f
```

4. Update (rebuild and restart):

```bash
docker compose up -d --build
```

5. Stop:

```bash
docker compose down
```

### Deploy to a remote server

Copy `docker-compose.yml`, `Dockerfile`, source files, and `.env` to the server, then run `docker compose up -d --build`.

Or transfer a pre-built image:

```bash
docker build --platform linux/amd64 -t bazaraki-notifier:amd64 .
docker save bazaraki-notifier:amd64 | ssh user@server "docker load"
```

Then on the server create a `docker-compose.yml` and `.env`, and run `docker compose up -d` (without `--build`).

### Persistent data

Subscription data is stored in the `bazaraki-data` Docker volume. It survives container restarts, stops, and re-creation. To reset all subscriptions:

```bash
docker volume rm bazaraki-data
```

## Run with Docker (without Compose)

```bash
docker build -t bazaraki-notifier .

docker run -d \
  --name bazaraki-notifier \
  --network host \
  --restart unless-stopped \
  -e TOKEN=your_telegram_bot_token \
  -e CHECKING_INTERVAL=300 \
  -e BATCH_SIZE=20 \
  -v bazaraki-data:/app/data \
  bazaraki-notifier
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
