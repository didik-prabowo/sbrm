# sbrm

Backend booking ruang meeting. Go + Echo + PostgreSQL.

## Jalanin

Pakai docker:

```
docker compose up -d --build
curl localhost:8080/healthz
```

Atau lokal (db tetap dari compose):

```
docker compose up -d db
cp .env.example .env
go run ./cmd/sbrm
```

Port lokal ikut `.env` (default contoh: 9000). Kalau `.env` tidak ada, config diambil dari env var `SBRM_*`.
Kalau `schema.sql` diubah, reset volume dulu: `docker compose down -v`.

Akun seed (password `password123`): `admin@test.com` (ADMIN), `user1@test.com`, `user2@test.com` (USER).

## Endpoint

Semua di bawah `/api/v1`, selain login wajib `Authorization: Bearer <token>`.

```
POST  /auth/login              {email, password}                         -> {token}
POST  /rooms                   {name, capacity}           (ADMIN)        -> 201
GET   /rooms/available?start=..&end=..                                   -> {data: [...]}
POST  /bookings                {room_id, title, start_time, end_time}    -> 201
PATCH /bookings/:id/cancel                                (pemilik/ADMIN)
```

Waktu pakai RFC3339, disimpan dalam UTC. Rentangnya `[start, end)`, jadi 09:00–10:00 dan 10:00–11:00 tidak bentrok.

Error selalu berbentuk `{"error_code": "...", "message": "..."}`, misal `ROOM_CONFLICT` (409), `INVALID_TIME` (400), `FORBIDDEN` (403).

Contoh:

```
TOKEN=$(curl -s localhost:8080/api/v1/auth/login -d '{"email":"user1@test.com","password":"password123"}' \
  -H 'Content-Type: application/json' | jq -r .token)

curl -s localhost:8080/api/v1/bookings -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"room_id":1,"title":"Weekly","start_time":"2026-12-01T09:00:00Z","end_time":"2026-12-01T10:00:00Z"}'
```

## Soal double booking

Create booking jalan di satu transaksi. Baris room di-lock dulu (`SELECT ... FOR UPDATE`) sehingga request ke room yang sama antre, baru cek overlap lalu insert.
Di level DB ada exclusion constraint (`btree_gist`) sebagai lapis kedua. Kalau constraint itu yang kena, API tetap balikin `409 ROOM_CONFLICT`.

## Struktur

```
cmd/sbrm            main
app/                wiring service http
internal/delivery   handler echo + middleware auth
internal/service    logic bisnis
internal/repository query postgres (raw sql)
internal/domain     entity & error
pkg/                jwt, bcrypt
```

## Test

```
go test ./pkg/...                                              # unit
BASE_URL=http://localhost:8080 go test ./tests/integration -count=1   # butuh server jalan
```

Integration test mencakup login, permission, aturan booking, cancel, dan 10 request paralel ke slot yang sama (harus 1 sukses, 9 conflict). Setiap test bikin room baru, jadi bisa diulang tanpa reset db.

Query analitik ada di `queries.sql`:

```
docker exec -i booking-db psql -U booking -d booking < queries.sql
```
