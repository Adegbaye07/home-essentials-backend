# Payments & webhooks

## Environment (backend)

| Variable | Description |
|----------|-------------|
| `PAYSTACK_SECRET_KEY` | `sk_test_…` / `sk_live_…` — server only |
| `SMTP_HOST` | e.g. `smtp.gmail.com` |
| `SMTP_PORT` | e.g. `587` |
| `SMTP_USER` | SMTP username (optional if host allows open relay) |
| `SMTP_PASSWORD` | SMTP password |
| `SMTP_FROM` | From address |
| `ADMIN_NOTIFY_EMAIL` | Inbox for new paid orders |
| `CLIENT_PUBLIC_URL` | Storefront base URL (emails/links), default `http://localhost:3000` |
| `PAYSTACK_CALLBACK_URL` | Post-payment redirect (default `{CLIENT_PUBLIC_URL}/store`) |

Client (`.env.local`):

| Variable | Description |
|----------|-------------|
| `NEXT_PUBLIC_API_URL` | Backend URL |
| `NEXT_PUBLIC_PAYSTACK_PUBLIC_KEY` | `pk_test_…` for InlineJS |

## Public catalogue

| Method | Path |
|--------|------|
| GET | `/api/v1/products` | Active products only. Query: `category` (optional), `page` (default 1), `page_size` (default 20, max 100). Response: `{ items, metadata }` |
| GET | `/api/v1/products/:id` | Active product detail |

## Checkout flow

Customer details and Paystack Inline run on the storefront **`/cart`** page (legacy `/checkout` redirects there).

1. `POST /api/v1/orders` — create order (`pending_payment`)
2. `POST /api/v1/payments/initialize` `{ "orderId": "..." }` → `{ "accessCode", "reference" }`
3. Client opens Paystack Inline with `accessCode`
4. Paystack `POST /api/v1/webhooks/paystack` — `charge.success`

### Cancel / abandon (customer)

When the customer closes Paystack without paying:

1. Client `POST /api/v1/payments/abandon` `{ "reference", "email" }` — email must match `order.customer.email` (case-insensitive).
2. Order moves from **`pending_payment`** → **`abandoned`**, sets `abandonedAt`, appends status history (“Payment cancelled by customer”).
3. Client redirects to `/checkout/success?reference=…&pending=1` (**cart is kept** so the customer can return to `/cart`).

Idempotent if already **`abandoned`** (200). Rejects **`paid`**+ with **409**. Wrong email → **403**.

### Resume payment

`POST /api/v1/payments/initialize` on an **`abandoned`** order:

- Sets status back to **`pending_payment`**, clears `abandonedAt`, appends “Payment resumed”, then Paystack initialize as usual.

The success page **Complete payment** button uses verify’s `orderId` + `customerEmail`, then initialize + Inline.

### Abandoned TTL (backend worker)

A background job runs every **3 minutes** and **hard-deletes** orders where `status == abandoned` and `abandonedAt` is older than **30 minutes**. After delete, verify/initialize return **404**; the client shows an “offer expired” message.

### Webhook status guard

- If order status is **`pending_payment`** or **`abandoned`**: set **`paid`**, append history, clear `abandonedAt`, set `paidAt`, assign `trackingNumber` if empty, send emails once. **Payment always wins** over abandon.
- If status is already **`paid`**, **`packing`**, **`in_transit`**, or **`delivered`**: **do not change status** (late webhook after manual admin updates).
- Idempotent email via `notifications.emailedAt`. Customer and admin receive **HTML** emails with line-item tables (SMTP required).

Email templates live under `internal/mail/email-templates/` (embedded at build time).

Configure webhook URL in Paystack dashboard (use ngrok/Cloudflare tunnel locally).

## Verify (success page)

`GET /api/v1/payments/verify?reference=fol_…`

Requires the Paystack **`reference`** returned from initialize (same value used on the success page query string). Response:

```json
{
  "orderId": "674a…",
  "status": "abandoned",
  "trackingNumber": "",
  "paystackStatus": "abandoned",
  "customerEmail": "customer@example.com"
}
```

- **`status`** — `pending_payment`, `abandoned`, or post-payment fulfillment states.
- **`customerEmail`** — checkout email on the order; used to reopen Paystack Inline when resuming payment.
- **`paystackStatus`** — from Paystack verify when configured; may be omitted if Paystack is not configured.
- Not a substitute for the payment webhook for marking orders paid.

## Admin order status

| From | Admin PATCH allowed |
|------|---------------------|
| `pending_payment` | → `paid` only (e.g. manual Paystack confirmation) |
| `abandoned` | **none** (read-only in admin UI; webhook or customer resume handles payment) |
| `paid`+ | Fulfillment transitions unchanged |

## Manual test checklist

Run backend (`air` / `go run`), client (`npm run dev`), admin (`npm run dev`), Paystack test keys, and webhook tunnel to `/api/v1/webhooks/paystack`.

- [ ] **Happy path:** Cart → fill details → **Pay with Paystack** → success page paid → tracking after webhook → cart empty.
- [ ] **Cancel checkout:** Close Paystack → **cart unchanged** → success `pending=1` → **Complete payment**, **Back to cart**, **Go to store** → verify shows `abandoned`.
- [ ] **Resume:** Complete payment → pay → success without `pending=1` → paid UI.
- [ ] **Cancel resume:** Complete payment → close Paystack → still unpaid; order `abandoned` again (optional: re-abandon API).
- [ ] **Admin:** Orders filter **Abandoned** → detail shows read-only status note; no Save status.
- [ ] **TTL (optional):** Wait 30m+ or lower TTL in dev → verify 404 → client “offer expired”.
- [ ] **Abandon API:** `curl -X POST …/payments/abandon -d '{"reference":"fol_…","email":"…"}'`
- [ ] **Verify API:** `curl -s "$API/api/v1/payments/verify?reference=fol_…" | jq`
- [ ] **Emails (optional):** SMTP + webhook → HTML customer/admin mail once.

Build verification:

```bash
cd home-essentials-backend && go test ./...
cd home-essentials-client && npm run build
cd home-essentials-admin && npm run build
```

## Track

`POST /api/v1/orders/track`

```json
{
  "trackingNumber": "KAM-20260803-ABC123",
  "email": "customer@example.com"
}
```

Tracking ID and email must match the order. Tracking is assigned when the order becomes **`paid`**.
