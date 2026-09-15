# Product API (Phase 1)

Admin product routes require a JWT from login (see [admin-seed.md](admin-seed.md)):

```http
Authorization: Bearer <token>
```

Base path: `/api/v1/admin/products`

## Categories

`foot_mats` | `door_mats` | `center_mats` | `rugs` | `cleaning_essentials`

## Variants

Free-text labels (same idea as the old “colors” field). Each variant needs an image URL in `variantImages`.

## Pricing

Delivery is **not** a product field. Storefront copy is fixed: *1–3 business days (Mon–Sat)*.

### Mats / rugs (`foot_mats`, `door_mats`, `center_mats`, `rugs`)

`sizePricings` — one row per free-text size:

| Field | Meaning |
|-------|---------|
| `size` | e.g. `2 x 5 ft`, `60cm x 90cm` |
| `piecePriceKobo` | Price for 1 piece |
| `bundlePriceKobo` | Price for 1 bundle |
| `piecesPerBundle` | How many pieces are in one bundle (≥ 1) |

`cleaningPricing` must be omitted.

### Cleaning essentials

No sizes. Require `cleaningPricing`:

| Field | Meaning |
|-------|---------|
| `piecePriceKobo` | Price for 1 piece |
| `dozenPriceKobo` | Price for 1 dozen (**always 12 pieces**) |

`sizePricings` must be empty / omitted.

Amounts are **kobo**: ₦5,000 → `500000`.

## Upload product image

`POST /api/v1/admin/uploads`

- Header: `Authorization: Bearer <token>`
- Body: `multipart/form-data` with field **`file`**
- Response: `{ "url": "https://..." }`

Add returned URLs to `variantImages` on create/update.

## Create product (rug example)

`POST /api/v1/admin/products`

```json
{
  "title": "Living room rug",
  "description": "Soft center rug.",
  "category": "rugs",
  "variants": ["beige", "grey"],
  "variantImages": [
    { "variant": "beige", "imageUrl": "https://example.com/beige.jpg" },
    { "variant": "grey", "imageUrl": "https://example.com/grey.jpg" }
  ],
  "active": true,
  "sizePricings": [
    {
      "size": "2 x 5 ft",
      "piecePriceKobo": 500000,
      "bundlePriceKobo": 4500000,
      "piecesPerBundle": 10
    },
    {
      "size": "3 x 5 ft",
      "piecePriceKobo": 700000,
      "bundlePriceKobo": 6300000,
      "piecesPerBundle": 10
    }
  ]
}
```

**201** — created product (includes Mongo `id`, timestamps).

## Create product (cleaning example)

```json
{
  "title": "Floor mop",
  "description": "Standard mop head.",
  "category": "cleaning_essentials",
  "variants": ["standard"],
  "variantImages": [
    { "variant": "standard", "imageUrl": "https://example.com/mop.jpg" }
  ],
  "active": true,
  "cleaningPricing": {
    "piecePriceKobo": 200000,
    "dozenPriceKobo": 2000000
  }
}
```

## List products

`GET /api/v1/admin/products`

Query: `category`, `active` (`true`|`false`), `page`, `page_size`.

## Public catalogue

`GET /api/v1/products` — active only  
`GET /api/v1/products/:id` — active only (404 if inactive)

## Update / delete

`PUT /api/v1/admin/products/:id` — same body as create  
`DELETE /api/v1/admin/products/:id` — **204**; blocked with **409** if open orders reference the product

## Shop order lines (wired for new pricing)

`POST /api/v1/orders` line shape:

```json
{
  "productId": "...",
  "variant": "beige",
  "size": "2 x 5 ft",
  "unit": "piece",
  "quantity": 2
}
```

- Mats/rugs: `unit` is `piece` or `bundle`; `size` required  
- Cleaning: `unit` is `piece` or `dozen`; omit `size`  
- Prices are always resolved server-side from the product document
