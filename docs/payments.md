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

## Checkout flow

Customer details and Paystack Inline run on the storefront **`/cart`** page (legacy `/checkout` redirects there).

1. `POST /api/v1/orders` — create shop order (`pending_payment`, reference `kam_…`)
2. `POST /api/v1/payments/initialize` `{ "orderId": "..." }` → `{ "accessCode", "reference" }`
3. Client opens Paystack Inline with `accessCode`
4. On success: cart cleared → `/checkout/success?reference=…`
5. Mark paid via webhook **`charge.success`** and/or **verify** reconciliation (if Paystack already reports success)

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

### Paid transition + emails

- If order status is **`pending_payment`** or **`abandoned`**: set **`paid`**, append history, clear `abandonedAt`, set `paidAt`, assign `trackingNumber` (`KAM-…`) if empty, send emails once. **Payment always wins** over abandon.
- If status is already **`paid`**, **`packing`**, **`in_transit`**, or **`delivered`**: **do not change status** (late webhook after manual admin updates).
- Idempotent email via `notifications.emailedAt`. Customer and admin receive **HTML** emails with line-item tables (SMTP required).

Email templates live under `internal/mail/email-templates/` (embedded at build time).

Configure webhook URL in Paystack dashboard: `https://YOUR_HOST/api/v1/webhooks/paystack` (use ngrok/Cloudflare tunnel locally).

## Verify (success page)

`GET /api/v1/payments/verify?reference=kam_…`

Requires the Paystack **`reference`** returned from initialize (same value used on the success page query string).

When Paystack reports **`success`** and the order is still unpaid, verify **reconciles** the same way as the webhook (marks paid + emails). This covers delayed/missing webhooks in local/test setups.

Response:

```json
{
  "orderId": "674a…",
  "status": "paid",
  "trackingNumber": "KAM-20260915-ABC123",
  "paystackStatus": "success",
  "customerEmail": "customer@example.com"
}
```

## Track

`POST /api/v1/orders/track`

```json
{
  "trackingNumber": "KAM-20260915-ABC123",
  "email": "customer@example.com"
}
```

Email match is case-insensitive. Tracking is assigned when the order becomes **`paid`**. Response includes `items`, `statusHistory`, and `totalAmountKobo`.

## Manual test checklist

Run backend (`go run .`), client (`npm run dev`), admin (`npm run dev`), Paystack **test** keys, and optionally a webhook tunnel to `/api/v1/webhooks/paystack`.

- [ ] **Happy path:** Cart → details → **Pay with Paystack** → success → paid + `KAM-…` tracking → cart empty → track page works.
- [ ] **Cancel:** Close Paystack → cart unchanged → success `pending=1` → **Complete payment** or **Back to cart**.
- [ ] **Resume:** Complete payment → pay → paid UI.
- [ ] **Emails (optional):** SMTP configured → customer + admin HTML mail once.
- [ ] **Admin:** Mark packing → in_transit → delivered; customer gets status emails if SMTP on.

```bash
cd home-essentials-backend && go test ./...
cd home-essentials-client && npm run build
```
