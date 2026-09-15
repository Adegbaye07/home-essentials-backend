# Product API (Phase 1)

Admin product routes require a JWT from login (see [admin-seed.md](admin-seed.md)):

```http
Authorization: Bearer <token>
```

Base path: `/api/v1/admin/products`

## Categories

`cross_body` | `hobo` | `duffel` | `male_toilet` | `school` | `travel` | `laptop` | `purse` | `clutch` | `tote` | `shoulder` | `shopping` | `rope` | `satchel` | `jute` | `lunch_box` | `waist_purse` | `folder` | `pencil_case` | `hand_bag` | `flap_bag`

`mens` is a **legacy** value: still accepted on existing products, but not offered in new-product UI.

## Sizes

`S` | `M` | `L` | `XL` | `XXL`

## Tier rules (per size)

- Tiers must start at quantity **1**.
- Ranges must not overlap; each range starts at `previous.maxQty + 1`.
- The **last tier** may omit `maxQty` (unlimited quantity above the previous max) **or** set `maxQty` to cap the maximum order quantity for that size.
- At most one open-ended tier per size; if present, it must be the last tier.
- Amounts are **kobo** (`unitPriceKobo`): ₦15,000 → `1500000`.
- **`deliveryDays`** (required on every tier): estimated lead time in **calendar days** (integer ≥ 1). Used for storefront/admin display; checkout pricing ignores it.

Example tiers: 1 @ ₦15k, 2–5 @ ₦13k, 6–10 @ ₦10k, 11+ @ ₦8k:

```json
[
  { "minQty": 1, "maxQty": 1, "unitPriceKobo": 1500000, "deliveryDays": 14 },
  { "minQty": 2, "maxQty": 5, "unitPriceKobo": 1300000, "deliveryDays": 10 },
  { "minQty": 6, "maxQty": 10, "unitPriceKobo": 1000000, "deliveryDays": 7 },
  { "minQty": 11, "unitPriceKobo": 800000, "deliveryDays": 5 }
]
```

Example with a **capped** last tier (max order qty 10 for this size):

```json
[
  { "minQty": 1, "maxQty": 1, "unitPriceKobo": 1500000, "deliveryDays": 14 },
  { "minQty": 2, "maxQty": 5, "unitPriceKobo": 1300000, "deliveryDays": 10 },
  { "minQty": 6, "maxQty": 10, "unitPriceKobo": 1000000, "deliveryDays": 7 }
]
```

## Upload product image

`POST /api/v1/admin/uploads`

- Header: `Authorization: Bearer <token>`
- Body: `multipart/form-data` with field **`file`**
- Response: `{ "url": "https://..." }`

Add returned URLs to product `colorImages` on create/update (one URL per color).

## Create product

`POST /api/v1/admin/products`

```json
{
  "title": "Classic Tote",
  "description": "Everyday canvas tote.",
  "category": "tote",
  "colors": ["black", "tan"],
  "colorImages": [
    { "color": "black", "imageUrl": "https://example.com/tote-black.jpg" },
    { "color": "tan", "imageUrl": "https://example.com/tote-tan.jpg" }
  ],
  "active": true,
  "sizes": [
    {
      "code": "M",
      "tiers": [
        { "minQty": 1, "maxQty": 1, "unitPriceKobo": 1500000, "deliveryDays": 14 },
        { "minQty": 2, "maxQty": 5, "unitPriceKobo": 1300000, "deliveryDays": 10 },
        { "minQty": 6, "unitPriceKobo": 1000000, "deliveryDays": 7 }
      ]
    },
    {
      "code": "L",
      "tiers": [
        { "minQty": 1, "maxQty": 1, "unitPriceKobo": 1600000, "deliveryDays": 14 },
        { "minQty": 2, "unitPriceKobo": 1200000, "deliveryDays": 7 }
      ]
    }
  ]
}
```

**201** — created product document (includes Mongo `id`, timestamps).

## List products

`GET /api/v1/admin/products`

Query:

- `category` — optional filter
- `active` — `true` | `false`
- `page` — optional, default `1`
- `page_size` — optional, default `20`, max `100`

**200** — paginated list:

```json
{
  "items": [ "...product..." ],
  "metadata": {
    "total_items": 42,
    "current_items": 20,
    "current_page": 1,
    "last_page": 3,
    "next_page": 2,
    "previous_page": null,
    "has_next_page": true,
    "has_previous_page": false
  }
}
```

## Get product

`GET /api/v1/admin/products/:id`

`:id` — 24-char hex ObjectId.

## Update product

`PUT /api/v1/admin/products/:id`

Same body as create.

## Delete product

`DELETE /api/v1/admin/products/:id`

`:id` — 24-char hex ObjectId.

**204** — product removed from MongoDB; all `colorImages` URLs are deleted from object storage when configured (failures are logged, not returned to the client).

**404** — product not found.

**409** — delete blocked because an order still references this product and is not `delivered`:

```json
{ "error": "There is a pending order for this item.", "code": "product_delete_pending_order" }
```

```json
{ "error": "There is an incomplete order for this item.", "code": "product_delete_incomplete_order" }
```

Pending payment is reported when any blocking order is in `pending_payment`; otherwise open orders (`paid`, `packing`, `in_transit`, etc.) use the incomplete message.

## Example: overlapping tiers (400)

```json
{
  "title": "Bad tiers",
  "description": "Test",
  "category": "tote",
  "colors": ["black"],
  "colorImages": [{ "color": "black", "imageUrl": "https://example.com/black.jpg" }],
  "active": true,
  "sizes": [
    {
      "code": "M",
      "tiers": [
        { "minQty": 1, "maxQty": 5, "unitPriceKobo": 100, "deliveryDays": 7 },
        { "minQty": 3, "maxQty": 10, "unitPriceKobo": 90, "deliveryDays": 7 }
      ]
    }
  ]
}
```

Returns **400** with overlap error.
