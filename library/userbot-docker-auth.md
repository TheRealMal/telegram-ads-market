# Userbot: first-time auth in Docker

The userbot uses Telegram’s interactive auth (phone number + code). In a normal container there is no TTY, so it gets **EOF** when reading input and fails with:

```text
Error: userbot start: callback: auth: auth flow: get code: EOF
```

**Fix:** run the userbot **once** with an interactive TTY so you can enter phone and code. The session is then stored in the `userbot_sessions` volume and reused by the container.

### One-time interactive login

1. Start the rest of the stack (so postgres/redis are up):
   ```bash
   docker compose -f docker-compose.https-selfsigned.yml up -d postgres redis
   ```

2. Run userbot interactively (same compose file and volume):
   ```bash
   docker compose -f docker-compose.https-selfsigned.yml run --rm -it userbot
   ```

3. When prompted, enter:
   - Your Telegram **phone number** (with country code, e.g. `+1234567890`)
   - The **login code** Telegram sends to your account

4. After a successful login, the session is written to the volume. Exit the container (Ctrl+C or type exit).

5. Start the full stack; userbot will use the saved session and no longer ask for auth:
   ```bash
   docker compose -f docker-compose.https-selfsigned.yml up -d
   ```

Ensure `.env` has `USER_BOT_API_ID`, `USER_BOT_API_HASH`, and `USER_BOT_PHONE` (optional; you can enter the phone in the prompt if not set).
