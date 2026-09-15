# Orders API (Phase 3)

Amounts are **kobo** (`totalAmountKobo`, line fields).

**Statuses:** `pending_payment` (on create, before Paystack) → **`paid`** (Paystack webhook in Phase 4, or manual admin if confirmed in Paystack dashboard) → **`packing`** → **`in_transit`** → **`delivered`**.

Admin PATCH enforces allowed transitions (e.g. from `pending_payment` only to `paid`).

## Create order (public)

`POST /api/v1/orders`

No auth. Server loads active products, applies **per-size quantity tiers**, snapshots line prices, sets `pending_payment`, and assigns `paystackReference` (`fol_…`) for Phase 4 Paystack.

```json
{
  "items": [
    {
      "productId": "674a1b2c3d4e5f6789012345",
      "size": "M",
      "color": "black",
      "quantity": 2
    }
  ],
  "customer": {
    "name": "John Doe",
    "email": "customer@example.com",
    "phone": "+2348000000000",
    "deliveryAddress": "12 Example St, Lagos"
  }
}
```

**201** — full order document (`items` include `productTitle`, `unitPriceKobo`, `lineTotalKobo`; `statusHistory` starts with `pending_payment`).

**400** — validation (inactive product, bad size/color, tier mismatch, etc.).

## Admin orders

Requires JWT (`Authorization: Bearer <token>`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/orders` | List newest first. Query: `status` (optional), `page` (default 1), `page_size` (default 20, max 100) |
| GET | `/api/v1/admin/orders/:id` | Order detail |
| PATCH | `/api/v1/admin/orders/:id/status` | Update status; appends `statusHistory` |
| DELETE | `/api/v1/admin/orders/:id` | Hard-delete **`abandoned`** orders only (**400** otherwise) |

**200** list response — same paginated shape as products:

```json
{
  "items": [ "...order..." ],
  "metadata": {
    "total_items": 120,
    "current_items": 20,
    "current_page": 1,
    "last_page": 6,
    "next_page": 2,
    "previous_page": null,
    "has_next_page": true,
    "has_previous_page": false
  }
}
```

List filter `status` values match admin UI filters (e.g. `paid`, `packing`, `abandoned`, `pending_payment`).

### Update status

```json
{
  "status": "packing",
  "note": "Items packed"
}
```

Valid admin targets: `paid`, `packing`, `in_transit`, `delivered` (from `pending_payment`, only `paid`).

**200** — updated order. **400** if transition not allowed or invalid status.

Admin status changes email the **customer** (HTML with status summary) when SMTP is configured. Payment webhook sends separate **HTML** confirmation emails (customer + admin, with line-item tables) once (`notifications.emailedAt`).

## Example: create then list (curl)

```bash
# Create (replace productId)
curl -s -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"productId":"YOUR_ID","size":"M","color":"black","quantity":1}],"customer":{"name":"Ada","email":"a@b.com","phone":"0800","deliveryAddress":"Addr"}}'

# Admin list (paginated)
curl -s 'http://localhost:8080/api/v1/admin/orders?page=1&page_size=20' \
  -H "Authorization: Bearer TOKEN" | jq '.metadata'

curl -s 'http://localhost:8080/api/v1/admin/orders?status=paid&page=1' \
  -H "Authorization: Bearer TOKEN" | jq .
```
