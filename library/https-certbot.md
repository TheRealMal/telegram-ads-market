# HTTPS with Certbot (Let's Encrypt)

The stack uses **Nginx** as reverse proxy and **Certbot** for Let's Encrypt certificates. Caddy has been replaced with this setup.

## Requirements

- A **real domain** with DNS A record pointing to this host (Let's Encrypt does not issue for IPs or localhost).
- Ports **80** and **443** available.

## Setup

1. **Set in `.env`:**
   - `DOMAIN=yourdomain.com`
   - `CERTBOT_EMAIL=you@example.com`
   - `CLIENT_DOMAIN=https://yourdomain.com` (for API redirects/links)

2. **Obtain certificate (run once; port 80 must be free):**
   ```bash
   docker compose -f docker-compose.https-selfsigned.yml run --rm certbot
   ```

3. **Start the stack:**
   ```bash
   docker compose -f docker-compose.https-selfsigned.yml up -d
   ```
   Nginx will wait for the certificate files to exist before starting (obtain cert first).

4. **Access:** `https://<DOMAIN>` — frontend at `/`, API at `/api/v1/*`.

## Renewal

Let's Encrypt certs are valid 90 days. Renew with webroot (nginx keeps running):

```bash
docker compose -f docker-compose.https-selfsigned.yml run --rm certbot renew --webroot -w /var/www/certbot
```

Then reload nginx to use the new cert:

```bash
docker compose -f docker-compose.https-selfsigned.yml exec nginx nginx -s reload
```

Add a cron job (e.g. monthly) to run the renew command and reload.

## Routing

- **Nginx** listens on 80 (ACME challenge + redirect to HTTPS) and 443 (TLS).
- Path-based: `/api/v1/*` → api:8080, everything else → web:3000.
- Certificates live in volume `certbot_letsencrypt` (from Certbot).

## Self-signed (no domain)

If you need HTTPS without a domain (e.g. local/dev), use the base `docker-compose.yml` or keep a separate Caddy-based compose with `tls internal` (self-signed). This Certbot stack is for production with a real domain.
