# Home Essentials by Kamgol — API

Go (Gin) backend for **Home Essentials by Kamgol** — MongoDB, Paystack, SMTP, and Supabase Storage.

## Phase 1 status

Product domain uses Home Essentials categories, variants + images, free-text size piece/bundle pricing, and cleaning piece/dozen pricing. Auth, uploads, and public catalogue are ready. Admin UI (Phase 2) still shows the old bag form until updated.

## Prerequisites

- Go 1.22+
- MongoDB (Atlas or local)

## Environment

Copy [`.env.example`](.env.example) to `.env` and fill in values (never commit `.env`).

| Variable | Required | Description |
|----------|----------|-------------|
| `HTTP_ADDR` | no | Listen address (default `:8080`) |
| `MONGODB_URI` | **yes** | Mongo connection string (include DB name path, e.g. `.../home_essentials`) |
| `CORS_ORIGINS` | no | Comma-separated origins (default `http://localhost:3000,http://localhost:3001`) |
| `JWT_SECRET` | **yes** for login | Secret for admin JWT |
| `SUPABASE_URL` | for uploads | Supabase project URL |
| `SUPABASE_SERVICE_ROLE_KEY` | for uploads | Legacy `eyJ…` service_role JWT |
| `SUPABASE_STORAGE_BUCKET` | for uploads | Public bucket name |
| `PAYSTACK_SECRET_KEY` | for payments | `sk_test_…` / `sk_live_…` |
| `SMTP_*` / `ADMIN_NOTIFY_EMAIL` | for email | Optional in early phases |
| `CLIENT_PUBLIC_URL` | no | Storefront URL (default `http://localhost:3000`) |
| `PAYSTACK_CALLBACK_URL` | no | Defaults to `{CLIENT_PUBLIC_URL}/store` |
| `APP_LOG_FILE` | no | Optional log file path |

## Run locally

```bash
cp .env.example .env
# edit .env
go mod tidy
go run .
```

- [http://localhost:8080/health](http://localhost:8080/health)
- [http://localhost:8080/ready](http://localhost:8080/ready)

Seed an admin user — see [docs/admin-seed.md](docs/admin-seed.md).
