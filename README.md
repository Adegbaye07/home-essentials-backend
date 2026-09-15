# Home Essentials by Kamgol — API

Go (Gin) backend for **Home Essentials by Kamgol** — MongoDB, Paystack, SMTP, and Supabase Storage.

Shop-only catalogue and orders (variants, free-text sizes, piece/bundle/dozen pricing, `KAM-` tracking, gold brand emails).

## Prerequisites

- Go 1.22+
- MongoDB (Atlas or local)

## Environment

Copy [`.env.example`](.env.example) to `.env` and fill in values (never commit `.env`).

| Variable | Required | Description |
|----------|----------|-------------|
| `HTTP_ADDR` | no | Listen address (default `:8080`) |
| `MONGODB_URI` | **yes** | Mongo URI including DB name (e.g. `.../home_essentials`) |
| `CORS_ORIGINS` | no | Comma-separated origins (default `http://localhost:3000,http://localhost:3001`) |
| `JWT_SECRET` | **yes** for login | Secret for admin JWT |
| `SUPABASE_URL` | for uploads | Supabase project URL |
| `SUPABASE_SERVICE_ROLE_KEY` | for uploads | Legacy `eyJ…` **service_role** JWT (not `sb_secret_…`) |
| `SUPABASE_STORAGE_BUCKET` | for uploads | Public bucket name |
| `PAYSTACK_SECRET_KEY` | for payments | `sk_test_…` / `sk_live_…` |
| `CLIENT_PUBLIC_URL` | no | Storefront URL for email links (default `http://localhost:3000`) |
| `PAYSTACK_CALLBACK_URL` | no | Defaults to `{CLIENT_PUBLIC_URL}/store` |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASSWORD` / `SMTP_FROM` | for email | Outbound mail |
| `ADMIN_NOTIFY_EMAIL` | for email | Inbox for paid-order alerts |
| `APP_LOG_FILE` | no | Optional log file path |

### Production CORS

Set `CORS_ORIGINS` to your live storefront and admin origins, e.g.:

```text
CORS_ORIGINS=https://shop.example.com,https://admin.example.com
```

No trailing slashes. Restart the API after changing env.

### Paystack webhook

In the [Paystack dashboard](https://dashboard.paystack.com/#/settings/developer):

1. Add webhook URL: `https://YOUR_API_HOST/api/v1/webhooks/paystack`
2. Use the same secret key as `PAYSTACK_SECRET_KEY`
3. Locally: tunnel (ngrok / Cloudflare) to `http://localhost:8080`

Verify also reconciles paid orders if the webhook is delayed (see [docs/payments.md](docs/payments.md)).

## Run locally

```bash
cp .env.example .env
# edit .env
go mod tidy
go run .
```

- Health: [http://localhost:8080/health](http://localhost:8080/health)
- Ready: [http://localhost:8080/ready](http://localhost:8080/ready)

Seed an admin — [docs/admin-seed.md](docs/admin-seed.md).

## Docs

| Doc | Topic |
|-----|--------|
| [docs/products.md](docs/products.md) | Product API |
| [docs/orders.md](docs/orders.md) | Orders API |
| [docs/payments.md](docs/payments.md) | Paystack, abandon, verify, track |
| [docs/supabase-storage.md](docs/supabase-storage.md) | Image uploads |
| [docs/smoke-checklist.md](docs/smoke-checklist.md) | End-to-end handoff checklist |

## Tests

```bash
go test ./...
```
