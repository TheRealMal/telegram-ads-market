# Ads Market — Telegram Advertising Marketplace

Telegram mini app providing smooth lessee & lessors interaction flow with automatic channels stats fetching, easy listings creation and transparent deals pipeline.

# MVP

* May be inaccessible without VPN *
- Bot: https://t.me/BaboonMarketBot
- Mini APP: https://t.me/BaboonMarketBot?startapp=
- Analytics dashboard: https://t.me/BaboonMarketBot/dashboard (admins only)

# Features
- Automatic channel fetching stats once bot added as an admin. Stats become available to all users when listing for channel is created
- Listing creation by any admin & owner of a channel
- Listing price range per any amount of hours
- Deal approvement when both sides sign a deal, automatic escrow wallet generation and deposit monitoring
- Automatic advertisment post message sending into a channel, every hour check that post is not deleted
- Automatic escrow release if advertisment conditions are met otherwise escrow refund to a lessee
- Chat with the other side of a deal via native telegram chat
- Event & workers driven architecture with locks & recovery for message posting & escrow releases
- Analytics shapshots being taken every day. Analytics dashboard available only to admins via another mini app direct link.
    1. Current listings, deals, users amount
    2. Total commission earned
    3. Average listings amount by user
    4. Pie chart with deals & deals amount by status
    5. Graph of 1, 2, 3 values over time


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

Example environment variables file presented at [example.env](example.env)

MVP can be deployed via make commands

```console
# Initialize vault
make init_vault

# Then you need to unseal it with 3 keys (use keys that were displayed)
make unseal_vault <KEY>

# And create secret storage (use token that was displayed)
make create_secret_vault <TOKEN>

# Initialize userbot session
make init_userbot

# Start
make start
```

## Frontend

Frontend sources are placed into `web` directory, fully written with AI on React + Next.js

# Preview

## Stats

<div style="display: flex; flex-direction: row; justify-content: center; align-items: center;">
    <div style="width: 90%; max-width: 90%; margin: 0 auto; display: flex; justify-content: center; align-items: center; overflow: hidden;">
        <img
            title="Stats"
            alt="Stats"
            src="/assets/stats.gif"
            style="
                width: 100%;
                max-width: 100%;
                display: block;
                object-fit: cover;
                object-position: center;
                /* Show only 20% vertical slice in the center */
                clip-path: inset(0 37% 0 37%);
            "
        />
    </div>
</div>


## Tabs
<div style="display: flex; flex-direction: row; gap: 8px; justify-content: center; align-items: center;">
    <img title="Market" alt="Market" src="/assets/page_market.png" style="width: 25%; max-width: 25%;" />
    <img title="Listings" alt="Listings" src="/assets/page_listings.png" style="width: 25%; max-width: 25%;" />
    <img title="Deals" alt="Deals" src="/assets/page_deals.png" style="width: 25%; max-width: 25%;" />
    <img title="Profile" alt="Profile" src="/assets/page_profile.png" style="width: 25%; max-width: 25%;" />
</div>

## Listing details & creation

<div style="display: flex; flex-direction: row; gap: 8px; justify-content: center; align-items: center;">
    <img title="Listing details" alt="Listing details" src="/assets/page_listing-details.png" style="width: 25%; max-width: 25%;" />
    <img title="Listing creation 1" alt="Listing creation 1" src="/assets/page_create-listing_1.png" style="width: 25%; max-width: 25%; margin-left: 5%;" />
    <img title="Listing creation 2" alt="Listing creation 2" src="/assets/page_create-listing_2.png" style="width: 25%; max-width: 25%;" />
</div>

## Deal details 

<div style="display: flex; flex-direction: row; gap: 8px; justify-content: center; align-items: center;">
    <img title="Deal details" alt="Deal details" src="/assets/page_deal-details.png" style="width: 25%; max-width: 25%; margin-left: 5%;" />
    <img title="Deal chat" alt="Deal chat" src="/assets/page_deal-chat.png" style="width: 25%; max-width: 25%;" />
</div>

# TODO

- [x] Fix deals being available to everyone (must be available only to deal sides)
- [x] Fetch channel picture
- [x] Secure escrow keys in database
- [ ] Somehow monitor removing & adding new admins which will have ability to create listings
- [x] Refresh channels stats & display last update date on stats page
- [x] Add telegram notifications about deals
- [ ] Add more deal options:
    - [ ] Keep post in top 1 (no more posts after it)
    - [ ] Story post
    - [ ] Add images to posts
    - [ ] Preview post in telegram messages button
- [ ] Send collected comission to treasury inside escrow release tx
- [ ] Collect escrow release time, incoming and outgoing transactions (db schema ready)