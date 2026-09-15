# Orders API (Phase 3)

Shop checkout only. Amounts are **kobo** (`totalAmountKobo`, line fields).

**Statuses:** `pending_payment` (on create) → **`paid`** (admin mark paid until Paystack webhook in a later phase) → **`packing`** → **`in_transit`** → **`delivered`**.

Customers may abandon unpaid checkouts (`abandoned`); those can be deleted by admin. There is no custom/recreate order flow.

Admin PATCH enforces allowed transitions (from `pending_payment` only to `paid`).

## Create order (public)

`POST /api/v1/orders`

No auth. Server loads active products, resolves **variant + size + unit** prices (`piece` / `bundle` for mats & rugs; `piece` / `dozen` for cleaning), snapshots line prices, sets `pending_payment`, and assigns `paystackReference` (`kam_…`) for later Paystack.

```json
{
  "items": [
    {
      "productId": "674a1b2c3d4e5f6789012345",
      "variant": "Beige",
      "size": "2 x 5 ft",
      "unit": "piece",
      "quantity": 2
    },
    {
      "productId": "674a1b2c3d4e5f6789012346",
      "variant": "Lemon",
      "unit": "dozen",
      "quantity": 1
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

**201** — full order document (`items` include `productTitle`, `unitPriceKobo`, `lineTotalKobo`, optional `piecesPerBundle`; `statusHistory` starts with `pending_payment`; `orderType` is `shop`).

**400** — validation (inactive product, bad variant/size/unit, cleaning with size, mats without size, etc.).

## Admin orders

Requires JWT (`Authorization: Bearer <token>`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/orders` | List newest first. Query: `status` (optional), `page` (default 1), `page_size` (default 20, max 100) |
| GET | `/api/v1/admin/orders/:id` | Order detail |
| PATCH | `/api/v1/admin/orders/:id/status` | Update status; appends `statusHistory`; sets `trackingNumber` (`KAM-…`) when moving to `paid` |
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

List filter `status` values: `pending_payment`, `abandoned`, `paid`, `packing`, `in_transit`, `delivered` (plus legacy `created` / `rejected` if any remain in the DB).

### Update status

```json
{
  "status": "packing",
  "note": "Items packed"
}
```

Valid admin targets: `paid`, `packing`, `in_transit`, `delivered` (from `pending_payment`, only `paid`).

**200** — updated order. **400** if transition not allowed or invalid status.

Admin status changes email the **customer** (HTML with status summary) when SMTP is configured.

## Example: create then mark paid (curl)

```bash
# Create (replace productId / variant / size to match an active product)
curl -s -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{"items":[{"productId":"YOUR_ID","variant":"Beige","size":"2 x 5 ft","unit":"piece","quantity":1}],"customer":{"name":"Ada","email":"a@b.com","phone":"0800","deliveryAddress":"Addr"}}'

# Admin list
curl -s 'http://localhost:8080/api/v1/admin/orders?page=1&page_size=20' \
  -H "Authorization: Bearer TOKEN" | jq '.metadata'

# Mark paid (replace ORDER_ID)
curl -s -X PATCH "http://localhost:8080/api/v1/admin/orders/ORDER_ID/status" \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"paid","note":"Confirmed offline"}' | jq '{status,trackingNumber,totalAmountKobo}'
```
