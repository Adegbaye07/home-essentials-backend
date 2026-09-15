# Admin user (manual seed)

Insert one document into MongoDB collection **`admins`**.

## Fields

| Field | Description |
|-------|-------------|
| `email` | Login email (unique) |
| `passwordHash` | bcrypt hash of password — never store plaintext |
| `role` | `"admin"` for dashboard access, or `"user"` (cannot use admin API) |
| `createdAt` | UTC date |

## Example document

```json
{
  "email": "admin@kamgol.com",
  "passwordHash": "$2a$12$................................................",
  "role": "admin",
  "createdAt": { "$date": "2026-08-03T00:00:00.000Z" }
}
```

Generate `passwordHash` with bcrypt (cost 10–12). Only users with **`role: "admin"`** receive a JWT from `POST /api/v1/admin/login` and can call protected admin routes.

## Login

```http
POST /api/v1/admin/login
Content-Type: application/json

{"email":"admin@kamgol.com","password":"your-plain-password"}
```

Response includes `token`. Use on product routes:

```http
Authorization: Bearer <token>
```

The middleware checks the JWT and that **`role` in the token is `admin`** (copied from the DB at login).
