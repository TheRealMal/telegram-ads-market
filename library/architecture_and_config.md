# Architecture, Config & Dev Workflow

Notes and decisions from implementation: config consolidation, Redis, env handling, and dev commands.

**Related:** `mvp_design.md` (marketplace product & system design), `monitoring_termnal_user_channel.md` (user channel monitoring).

---

## 1. Config architecture

### Single Telegram config

- **Decision:** Use only `internal/helpers/telegram/config` for bot-related settings. Remove `internal/bot/config`.
- **Root config** (`internal/config/config.go`): has `Telegram telegramconfig.Config` (no `Bot` field).
- **Bot command** (`cmd/bot/main.go`): builds the Telegram API client from `cfg.Telegram` only (no mix of Auth + Bot config).
- **Env vars** for the bot come from the telegram helper config: `BOT_TOKEN`, `BOT_USERNAME`, `BOT_WEB_APP_NAME`, `SECRET_TOKEN`, `RATE_LIMIT`.

### Env loading

- **Mechanism:** `internal/config/load.go` uses `godotenv.Load(envFile)` then `cleanenv.ReadEnv(&cfg)`.
- **Default file:** `.env`. Override via `ENV_FILE` (e.g. `ENV_FILE=.env`).
- **Template:** `example.env` lists all env vars with comments; copy to `.env` and fill secrets. Do not commit `.env` (ignored via `.gitignore`).

---

## 2. Redis config

- **Source:** Replaced single-`Addr` config with a host/port/auth/TLS config aligned with notway-backend style.
- **Location:** `internal/redis/config/config.go`.

**Fields:**

| Field          | Env                     | Default   | Notes                    |
|----------------|-------------------------|-----------|--------------------------|
| Host           | REDIS_HOST              | 0.0.0.0   |                          |
| Port           | REDIS_PORT              | 6379      |                          |
| User           | REDIS_USER              | ""        | Optional                 |
| Password       | REDIS_PASSWORD          | ""        | Optional                 |
| DB             | REDIS_DB                | 0         |                          |
| MinTLSVersion  | REDIS_MIN_TLS_VERSION   | 769       | TLS 1.0                  |
| EnableTLS      | REDIS_ENABLE_TLS        | false     | Set true for production  |

- **Client** (`internal/redis/client.go`): `optsFromConfig(cfg)` builds `redis.Options`:
  - `Addr`: `net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))`
  - `Username`, `Password`, `DB` from config
  - If `EnableTLS`: `TLSConfig` with `MinVersion: cfg.MinTLSVersion`
- **Usage:** All callers keep passing `cfg.Redis` into `redis.New` / `redis.NewOptional`; no other code changes.

---

## 3. Repository structure (reference)

- **cmd**
  - `main.go` — single entrypoint (Cobra).
  - `<service>/main.go` — Cobra command for that service (e.g. bot, market, userbot).
- **internal**
  - `config` — root config and load (env file, cleanenv).
  - `redis` — Redis client, config, implementation, stream.
  - `postgres` — Postgres client and config.
  - `helpers/telegram` — Telegram API client and config (single source for bot token, secret, rate limit, etc.).
  - `bot` — Telegram bot (webhook handler, updates service).
  - `userbot` — Telegram user bot (MTProto, monitoring).
  - `server` — HTTP server, routers, health, metrics.
  - `market` — Marketplace domain, application, repository.
  - `event` — Event domain and handlers (e.g. telegram_update, crypto_payment).
- **dockerfiles/dev** — Dev container Dockerfile and scripts.
- **docker-compose.yml** — Local Postgres, Redis, migrations.
- **library** — Architecture, concepts, and research notes.

---

## 4. Dev workflow (dev.sh & example.env)

### example.env

- One template with all env vars used by the app, grouped and commented.
- Sections: App, Database, Redis, Server, Health, Auth, Telegram bot, User bot, optional TON liteclient, Docker Compose overrides.
- **Usage:** Copy to `.env` and fill required values (e.g. `cp example.env .env`).

### dev.sh commands

- **ensure_env():** If `.env` is missing, copy `example.env` → `.env` and remind to edit; used by all run commands.
- **build:** Build dev container; if `.env` missing, create it from `example.env`.
- **run-bot:** Run Telegram bot (`cmd/bot`). Uses `ENV_FILE=.env`.
- **run-market:** Run market API (`cmd/market`). Uses `ENV_FILE=.env`.
- **run-userbot:** Run user bot (`cmd/userbot`). Uses `ENV_FILE=.env`.
- **run-all:** Start bot, market, and userbot in the background with `ENV_FILE=.env`, then wait.
- **start:** Start the dev container (interactive shell).
- **stop:** Stop the dev container.
- **shell:** Open a shell in the running dev container.
- **help | -h | --help:** Print usage (parsed from script comments).

### Typical usage

1. `./dev.sh build` — create `.env` from `example.env` if needed, build dev image.
2. Edit `.env` with secrets (BOT_TOKEN, TELEGRAM_BOT_TOKEN, DB_*, etc.).
3. Run services: `./dev.sh run-bot`, `./dev.sh run-market`, `./dev.sh run-userbot`, or `./dev.sh run-all`.

---

## 5. Concepts (short)

- **Config:** One source of truth per concern (e.g. one Telegram config in helpers/telegram/config; Redis config with host/port/auth/TLS in internal/redis/config).
- **Env:** One template (`example.env`), one active file (`.env`), optional override via `ENV_FILE`; app loads via godotenv + cleanenv.
- **Services:** Bot (webhook + updates), Market (API), Userbot (MTProto); can run separately or together via `run-all`.
