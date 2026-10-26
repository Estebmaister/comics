# Catalog Security Boundary

Catalog data is publicly readable. Catalog administration and scrape operations
require an access JWT in the `Authorization` header whose subject role is
`admin`.

Bearer-only admin operations reject browser cookies and never redirect to
`/login`; profile and dashboard routes may still use the HttpOnly access-token
cookie.

## Route policy

Public reads:

- `GET /comics`
- `GET /comics/:id`
- `GET /comics/search/:title`

Bearer-only admin operations:

- `POST /comics`
- `PUT /comics/:id`
- `DELETE /comics/:id`
- `PATCH /comics/:id/cover-visibility`
- `PATCH /comics/:id/:merging_id`
- `PUT /comics/:id/:merging_id`
- `GET /scrape`
- `GET /scrape/status`

Admin API requests must send `Accept: application/json` and an `Authorization`
header whose value is the word `Bearer`, one space, and the access JWT returned
by login.

The HttpOnly access-token cookie remains supported for browser profile and
dashboard routes. It is intentionally not accepted by catalog administration
routes. The React app reads its access JWT from `sessionStorage`, sends the
bearer header for admin operations, clears the session on `401`, and keeps a
valid session on `403`.

## Admin role migration

No SQL or MongoDB schema migration is required. Before deploying the protected
backend, promote one existing application user. Run these commands from the
repository root with a configured `go_server/.env`; the checked-in
`go_server/local.env` is only the local-development fallback:

```bash
: "${ENV_FILE:=go_server/.env}"
test -f "$ENV_FILE" || {
  echo "Create $ENV_FILE from go_server/local.env and set the target Mongo values" >&2
  exit 2
}
set -a
source "$ENV_FILE"
set +a
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to an existing application account email}"
export ADMIN_EMAIL

mongosh "$DB_ADDR" \
  --username "$DB_USER" \
  --password "$DB_PASS" \
  --authenticationDatabase admin \
  --quiet \
  --eval '
const databaseName = process.env.DB_NAME || "comics";
const collectionName = process.env.DB_TABLE_USERS || "users";
const email = process.env.ADMIN_EMAIL;
const users = db.getSiblingDB(databaseName).getCollection(collectionName);
const result = users.updateOne(
  { email },
  { $set: { role: "admin", active: true } }
);
if (result.matchedCount !== 1) {
  print(`expected one application user for ${email}; matched ${result.matchedCount}`);
  quit(2);
}
const doc = users.findOne({ email });
printjson({ email: doc.email, username: doc.username, role: doc.role, active: doc.active });
'
```

Expected output contains the selected email, `"role" : "admin"`, and
`"active" : true`. The promoted user must sign out and sign in again. Existing
access and refresh JWTs retain the role that was present when they were issued.

## Deployment order

1. Promote and verify one application admin.
2. Deploy the backend and frontend from the same security-boundary release.
3. Sign in again as the promoted admin.
4. Run the smoke checks below.
5. Keep the previous application image available until the checks pass.

There is no catalog downtime and no SQLite/PostgreSQL migration. Public read
URLs, query parameters, pagination headers, and payloads are unchanged.

## Compatibility

- Existing read-only clients continue to work without authentication.
- Existing write clients receive `401` until they send an admin bearer JWT.
- Authenticated non-admin clients receive `403`; their session remains valid.
- `GET /scrape` and `GET /scrape/status` keep their existing methods and
  response bodies but now require an admin bearer JWT.
- Browser cookie authentication remains compatible with `/protected/*` and
  `/admin/dashboard`.
- SQLite and PostgreSQL catalog behavior is unchanged.

## Smoke checks

Run against a local or staging deployment:

The commands require `curl`, `jq`, and `mongosh`.

```bash
: "${API_URL:=https://localhost:8081}"
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to the promoted application account email}"
: "${ADMIN_PASSWORD:?Set ADMIN_PASSWORD to that account password}"

TOKEN="$(
  curl -skS "$API_URL/login" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    --data "$(jq -nc \
      --arg email "$ADMIN_EMAIL" \
      --arg password "$ADMIN_PASSWORD" \
      '{email:$email,password:$password}')" |
  jq -er '.data.access_token'
)"

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    "$API_URL/comics?from=0&limit=1"
)" = 200

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    -X PUT "$API_URL/comics/2147483647" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    --data '{"track":false}'
)" = 401

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    "$API_URL/scrape/status" \
    -H 'Accept: application/json' \
    -H "Authorization: Bearer $TOKEN"
)" = 200

test "$(
  curl -skS -o /dev/null -w '%{http_code}' \
    -X PUT "$API_URL/comics/2147483647" \
    -H 'Accept: application/json' \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $TOKEN" \
    --data '{"track":false}'
)" = 404
```

The final `404` is expected: it proves the admin request passed both middleware
checks and reached the comic handler without changing a real record.

## Rollback

Prefer a frontend-only rollback while retaining the protected backend. Public
catalog reads continue to work; management remains unavailable until a
bearer-aware frontend is restored.

Rolling back the backend reintroduces public writes and public scrape triggers.
Before deploying an older backend, deny external `POST`, `PUT`, `PATCH`, and
`DELETE` requests under `/comics` and deny `GET /scrape` plus
`GET /scrape/status` at the ingress or reverse proxy. Remove those external
denies only after the protected backend is restored.

If the promoted account must be demoted after a full rollback, run:

```bash
: "${ENV_FILE:=go_server/.env}"
test -f "$ENV_FILE" || {
  echo "Create $ENV_FILE from go_server/local.env and set the target Mongo values" >&2
  exit 2
}
set -a
source "$ENV_FILE"
set +a
: "${ADMIN_EMAIL:?Set ADMIN_EMAIL to the account being demoted}"
export ADMIN_EMAIL

mongosh "$DB_ADDR" \
  --username "$DB_USER" \
  --password "$DB_PASS" \
  --authenticationDatabase admin \
  --quiet \
  --eval '
const databaseName = process.env.DB_NAME || "comics";
const collectionName = process.env.DB_TABLE_USERS || "users";
const email = process.env.ADMIN_EMAIL;
const result = db
  .getSiblingDB(databaseName)
  .getCollection(collectionName)
  .updateOne({ email }, { $set: { role: "user" } });
if (result.matchedCount !== 1) {
  print(`expected one application user for ${email}; matched ${result.matchedCount}`);
  quit(2);
}
printjson({ email, role: "user" });
'
```

No catalog backup or restore is needed for this authorization-only release.
