# Smoke checklist — Home Essentials by Kamgol

End-to-end handoff. Run backend (`:8080`), admin (`:3001`), and client (`:3000`) with real or test env values filled in.

## Preflight

- [ ] Backend `.env` from `.env.example` — `MONGODB_URI` includes `/home_essentials`, `JWT_SECRET` set
- [ ] Admin `src/lib/config.ts` — `apiUrl`
- [ ] Client `src/lib/config.ts` — `apiUrl` + `paystackPublicKey`
- [ ] `CORS_ORIGINS` includes both frontend origins
- [ ] `GET /health` and `GET /ready` OK
- [ ] Admin user seeded ([admin-seed.md](admin-seed.md))
- [ ] `cd home-essentials-backend && go test ./...` passes

## Phase 1 — Product images (1MB)

- [ ] Admin create/edit: helper text says **Maximum image size: 1MB**
- [ ] Image **≤1MB** → preview + save OK; image shows on storefront
- [ ] Image **>1MB** → toast “Image must be 1MB or smaller”; no preview
- [ ] Video still **≤5MB** OK / **>5MB** blocked in admin
- [ ] Optional API: `POST /api/v1/admin/uploads` with >1MB file → **400** `image exceeds 1MB limit`

## Phase 2 — Effective per-piece pricing

- [ ] Admin mat size: bundle ₦534,000 / 60 pcs → read-only **≈ ₦8,900** per piece; updates live when bundle or count changes
- [ ] Admin cleaning: dozen price → read-only **dozen ÷ 12** per piece
- [ ] Admin catalogue expand: “Per pc (bundle)” / “Per pc (dozen)” shown
- [ ] Store mat PDP: `bundle of N ₦… at ₦… per piece`
- [ ] Store cleaning PDP: dozen line includes `at ₦… per piece`
- [ ] Save/reload product — per-piece fields are derived only (not stored as API fields)

## Phase 3 — Delivery fee

- [ ] Piece-only cart → checkout Delivery **₦2,000**; Total = items + 2,000
- [ ] Add any **bundle** or **dozen** line → Delivery **Free**; Total = items only
- [ ] Cart + checkout show timeline + fee rule copy
- [ ] PDP shows fee rule under delivery timeline
- [ ] **Pay with Paystack** (test) → charged amount matches checkout total
- [ ] Admin order detail: Subtotal / Delivery / Total match payment
- [ ] `/track`: same Subtotal / Delivery / Total breakdown
- [ ] Optional SMTP: customer + admin paid emails show Subtotal / Delivery / Total
- [ ] Cancel Paystack → resume payment → amount unchanged (frozen at create)
- [ ] Legacy order (pre-fee) in admin → Delivery **Free**, total unchanged

## Catalogue (admin → store)

- [ ] Admin login works
- [ ] Create a **rug/mat** product: variants + images, ≥1 size with piece/bundle/`piecesPerBundle`, **active**
- [ ] Create a **cleaning** product: variants + images, piece + dozen prices, no sizes, **active**
- [ ] Store `/store` shows both; category tabs filter correctly
- [ ] PDP: variant swaps image; mat size + piece/bundle; cleaning piece/dozen; delivery timeline “1–3 business days (Mon–Sat)”

## Checkout (client → Paystack)

- [ ] Add piece + bundle (or dozen) lines; cart line prices match PDP math
- [ ] **Pay with Paystack** (test card) → success → cart empty → status **paid** + `KAM-…` tracking
- [ ] Optional: cancel Paystack → cart kept → `pending=1` → **Complete payment** resumes
- [ ] Optional: SMTP → customer + admin paid emails once

## Track + fulfil

- [ ] `/track` with tracking ID + checkout email → timeline + line items + amounts
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
