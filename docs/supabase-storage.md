# Supabase Storage (product images & videos)

The API uploads product images and authenticity videos server-side using the **service role** key. Never put Supabase secrets in the admin or client apps.

## Dashboard setup

1. In [Supabase](https://supabase.com/dashboard) → **Storage**, create a bucket (e.g. `product-images`).
2. Set the bucket to **public** so storefront and admin can load images via HTTPS URL.
3. Copy **Project URL** from **Project Settings → API**.
4. Copy the **service_role** key under **Project API keys** — the long JWT that starts with `eyJ`. Do **not** use the newer **Secret key** (`sb_secret_…`); [storage-go](https://github.com/supabase-community/storage-go) expects a JWT and uploads fail with `Invalid Compact JWS` if you paste the wrong format.

## Backend environment

Add to [`.env`](../.env) in the backend repo root (gitignored):

| Variable | Value |
|----------|--------|
| `SUPABASE_URL` | `https://<project-ref>.supabase.co` |
| `SUPABASE_SERVICE_ROLE_KEY` | **service_role** JWT (`eyJ…`), single line, no quotes |
| `SUPABASE_STORAGE_BUCKET` | Bucket name, e.g. `product-images` |

Restart the API from the **backend repo root** (`go run .` or `air`). On success you should see:

```text
supabase storage enabled bucket=product-images
```

## Upload flow

- Images: Admin UI → `POST /api/v1/admin/uploads` (JWT, multipart field `file`)
- Videos: Admin UI → `POST /api/v1/admin/uploads/video` (JWT, multipart field `file`)
- API stores objects under `products/{uuid}.{ext}` via [storage-go](https://github.com/supabase-community/storage-go)
- Response: `{ "url": "<public-url>" }`
  - Images → product `variantImages[].imageUrl`
  - Videos → product `videoUrl` (optional authenticity clip)

Allowed image types: JPEG, PNG, WebP, GIF (max **5MB**).  
Allowed video types: MP4, WebM, MOV (max **5MB**).

If env vars are missing, the routes still exist but return **503** `upload service not configured`.

## Orphan cleanup on product update

When an admin **updates** a product and removes or replaces a `variantImages` URL or `videoUrl`, the API deletes the old Supabase object **after** MongoDB saves successfully. Only paths under `products/` in your configured bucket are removed. Failed deletes are logged and do not fail the update. Product **delete** also removes stored images and video for that product.

Uploads that never get saved on a product (abandoned form) are **not** deleted automatically.

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `upload failed` / `Invalid Compact JWS` in `APP_LOG_FILE` | `SUPABASE_SERVICE_ROLE_KEY` is an `sb_secret_…` key or malformed — switch to the **service_role** JWT (`eyJ…`). |
| **503** `upload service not configured` | One of the three Supabase env vars is empty; restart after editing `.env`. |
| **500** / **502** `failed to upload sample image` / `upload failed` | Wrong bucket name, bucket not public, network, or `SUPABASE_URL` project ref no longer resolves (`NXDOMAIN`) — confirm Project URL in the dashboard and restart the API. |

## Verify

```bash
# After login, replace TOKEN and path to a small jpg / mp4
curl -s -X POST http://localhost:8080/api/v1/admin/uploads \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@./test.jpg"

curl -s -X POST http://localhost:8080/api/v1/admin/uploads/video \
  -H "Authorization: Bearer TOKEN" \
  -F "file=@./clip.mp4"
```

Open the returned `url` in a browser.
