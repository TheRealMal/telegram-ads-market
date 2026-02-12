# Ads Market — Telegram Advertising Marketplace

Telegram mini app providing smooth lessee & lessors interaction flow with automatic channels stats fetching, easy listings creation and transparent deals pipeline.

# MVP

- Bot: https://t.me/BaboonMarketBot
- Mini APP: https://t.me/BaboonMarketBot?startapp=

# Features
- Automatic channel fetching stats once bot added as an admin. Stats become available to all users when listing for channel is created
- Listing creation by any admin & owner of a channel
- Listing price range per any amount of hours
- Deal approvement when both sides sign a deal, automatic escrow wallet generation and deposit monitoring
- Automatic advertisment post message sending into a channel, every hour check that post is not deleted
- Automatic escrow release if advertisment conditions are met otherwise escrow refund to a lessee

# User flow
## Lessor flow
1. Connect a channel to mini app (add needed user as an admin, guide provided)
2. Create a listing with linked channel and price-duration ranges
3. Once lessee creates draft deal, you agree on deal details, then connect a wallet (for payment) and sign a deal

## Lessee flow
1. Create advertisment request listing
2. Anyone with connected channel to market can draft a deal
3. Lessee can view stats and agree on deal details, then connect a wallet (for possible refund) and sign a deal

## Shared flow
4. When both sides sign a deal new escrow address is being generated and lessee will be requested to make a safe deposit
5. System detect successful deposit and processes deal - posts message to lessor channel and stars monitoring it
6. When last check is done escrow funds are released to lessor. Otherwise escrow funds are refunded to a lessee

# Development

## Architecture

### Components 

- Postgres
- Redis
- Blockchain observer service
    - Monitors TON blockchain for escrow deposits. Notifies market about new escrow addresses deposits via redis stream.
- Bot service
    - Handles updates from a telegram by saving them in a redis stream and then processing via another worker. Example updates: /start command message, reply to a deal chatting message, etc.
- Userbot service
    - Monitors granting of admin rights in channels, fetches channels stats, posts advertisments messages and checks their existance.
- Market service
    - Exposes main API and handles all actions, creates escrow wallets, confirms deals after all checks passed.

## Deployment

MVP can be deployed via make command or directly via docker compose

```console
make start
```

```console
docker compose -f docker-compose.https-selfsigned.yml up -d
```

## Frontend

Frontend sources are placed into `web` directory, fully written with AI on React + Next.js