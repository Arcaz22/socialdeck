# Social Deck Backend

Backend Social Deck adalah API untuk game kartu sosial seperti Truth or Dare, Truth or Truth, dan Talk More. API ini menangani autentikasi, deck kartu, room game, state game, dan event realtime melalui WebSocket.

## Stack

- Go
- Gin HTTP framework
- PostgreSQL
- Redis
- Gorilla WebSocket
- Swagger docs

## Menjalankan Project

Jalankan database dan Redis:

```bash
docker compose up -d
```

Pastikan `.env` sudah terisi. Contoh nilai lokal:

```env
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=socialdeck
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
REDIS_URL=redis://localhost:6379/0
AUTH_SECRET_KEY=dev-secret
ACCESS_TOKEN_TTL_MINUTES=15
REFRESH_TOKEN_TTL_DAYS=30
```

Jalankan backend:

```bash
go run .
```

Default server berjalan di:

```txt
http://localhost:8080
```

Swagger tersedia di:

```txt
http://localhost:8080/docs/index.html
```

## Modul Utama

### Auth

Auth menyediakan register, login, refresh token, logout, dan current user.

Endpoint:

```txt
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

Login menghasilkan `access_token` yang dipakai sebagai Bearer token:

```txt
Authorization: Bearer <access_token>
```

### Deck

Deck adalah kumpulan kartu yang akan dipakai dalam game. Project ini sudah memiliki seed deck sistem:

```txt
00000000-0000-0000-0000-000000000001 - Truth or Dare Classic
00000000-0000-0000-0000-000000000002 - Truth or Truth
00000000-0000-0000-0000-000000000003 - Talk More
```

Endpoint:

```txt
GET    /api/v1/decks
GET    /api/v1/decks/me
GET    /api/v1/decks/:id
POST   /api/v1/decks
PUT    /api/v1/decks/:id
DELETE /api/v1/decks/:id
POST   /api/v1/decks/:id/cards
DELETE /api/v1/decks/:id/cards/:cardId
```

### Game Room

Room dibuat oleh host dari sebuah deck. Host otomatis menjadi player pertama. Player lain masuk memakai room code.

Endpoint REST game:

```txt
POST /api/v1/game/rooms
POST /api/v1/game/rooms/join
GET  /api/v1/game/rooms/:id/state
POST /api/v1/game/rooms/:id/leave
POST /api/v1/game/rooms/:id/ws-ticket
GET  /api/v1/game/rooms/:id/ws?ticket=<ticket>
```

## Alur Game

1. User register atau login.
2. Client mengambil deck publik dari `/api/v1/decks`.
3. Host membuat room dengan `deck_id`.
4. Backend mengembalikan `session_id` dan `code`.
5. Player lain join room memakai `code`.
6. Setiap player meminta WebSocket ticket melalui `/ws-ticket`.
7. Setiap player connect ke WebSocket memakai ticket.
8. Host mengirim event `start_game`.
9. Backend menentukan giliran pertama.
10. Player yang sedang giliran mengirim event `draw_card`.
11. Backend broadcast kartu ke semua player.
12. Player yang sama mengirim `submit_result` dengan `done` atau `pass`.
13. Backend menyimpan kartu yang sudah dimainkan dan memindahkan giliran.
14. Game selesai jika kartu habis atau host mengirim `stop_game`.

## WebSocket

WebSocket tidak memakai header `Authorization`. Auth dilakukan dengan short-lived ticket.

Ticket dibuat lewat REST:

```txt
POST /api/v1/game/rooms/:id/ws-ticket
Authorization: Bearer <access_token>
```

Ticket berlaku 30 detik dan sekali pakai.

Connect:

```txt
ws://localhost:8080/api/v1/game/rooms/:id/ws?ticket=<ticket>
```

Incoming event dari client ke server:

```json
{"type":"start_game","payload":{}}
{"type":"draw_card","payload":{}}
{"type":"submit_result","payload":{"result":"done"}}
{"type":"submit_result","payload":{"result":"pass"}}
{"type":"stop_game","payload":{}}
```

Outgoing event dari server:

```txt
player_joined
player_left
room_state
game_started
card_drawn
turn_result
turn_changed
game_finished
error
```

## Aturan Penting

- `start_game` hanya bisa dilakukan host.
- Minimal 2 player untuk memulai game.
- `draw_card` hanya bisa dilakukan player yang sedang giliran.
- `submit_result` hanya menerima `done` atau `pass`.
- Player yang belum join room tidak bisa membuat WebSocket ticket.
- Room waiting memiliki TTL 30 menit.
- WebSocket ticket expired dalam 30 detik dan hanya bisa dipakai sekali.

## Testing

Untuk testing WebSocket dari terminal, lihat:

[TESTING_WEBSOCKET.md](./TESTING_WEBSOCKET.md)
