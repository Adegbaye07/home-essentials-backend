# Smoke checklist — Home Essentials by Kamgol

End-to-end handoff. Run backend (`:8080`), admin (`:3001`), and client (`:3000`) with real or test env values filled in.

## Preflight

- [ ] Backend `.env` from `.env.example` — `MONGODB_URI` includes `/home_essentials`, `JWT_SECRET` set
- [ ] Admin `.env.local` — `NEXT_PUBLIC_API_URL`
- [ ] Client `.env.local` — `NEXT_PUBLIC_API_URL` + `NEXT_PUBLIC_PAYSTACK_PUBLIC_KEY`
- [ ] `CORS_ORIGINS` includes both frontend origins
- [ ] `GET /health` and `GET /ready` OK
- [ ] Admin user seeded ([admin-seed.md](admin-seed.md))

## Catalogue (admin → store)

- [ ] Admin login works
- [ ] Create a **rug/mat** product: variants + images, ≥1 size with piece/bundle/`piecesPerBundle`, **active**
- [ ] Create a **cleaning** product: variants + images, piece + dozen prices, no sizes, **active**
- [ ] Store `/store` shows both; category tabs filter correctly
- [ ] PDP: variant swaps image; mat size + piece/bundle; cleaning piece/dozen; delivery copy “1–3 business days (Mon–Sat)”

## Checkout (client → Paystack)

- [ ] Add piece + bundle (or dozen) lines; cart totals match PDP math
- [ ] **Pay with Paystack** (test card) → success → cart empty → status **paid** + `KAM-…` tracking
- [ ] Optional: cancel Paystack → cart kept → `pending=1` → **Complete payment** resumes
- [ ] Optional: SMTP → customer + admin paid emails once

## Track + fulfil

- [ ] `/track` with tracking ID + checkout email → timeline + line items
- [ ] Admin: paid → packing → in_transit → delivered (status emails if SMTP on)
- [ ] Abandoned order appears read-only in admin; delete works; TTL removes after ~30 minutes

## Production notes

- [ ] Live `CORS_ORIGINS`, `CLIENT_PUBLIC_URL`, Paystack **live** keys
- [ ] Paystack webhook → `https://API/api/v1/webhooks/paystack`
- [ ] Contact placeholders updated in `home-essentials-client/src/lib/contact.ts`
- [ ] Supabase bucket public; service_role JWT configured for uploads

## Quick commands

```bash
# API
cd home-essentials-backend && go test ./... && go run .

# Admin
cd home-essentials-admin && npm run build && npm run dev

# Client
cd home-essentials-client && npm run build && npm run dev
```
