# Testing WebSocket Game Dari Terminal

Dokumen ini menjelaskan cara test alur game Social Deck dari terminal tanpa Swagger. Swagger bisa dipakai untuk REST biasa, tetapi WebSocket lebih mudah dites lewat script terminal.

Script yang dipakai:

```txt
game-ws-test.sh
```

Script ini memakai client Go kecil di:

```txt
cmd/wscli/main.go
```

Jadi tidak perlu install `websocat`.

## 1. Jalankan Backend

Pastikan Postgres dan Redis hidup:

```bash
docker compose up -d
```

Jalankan backend:

```bash
go run .
```

## 2. Siapkan Token, Session, dan Room Code

Edit bagian atas `game-ws-test.sh`:

```bash
BASE="${BASE:-http://localhost:8080/api/v1}"
SESSION_ID="${SESSION_ID:-ganti_session_id}"
ROOM_CODE="${ROOM_CODE:-ganti_room_code}"

P1_TOKEN="${P1_TOKEN:-ganti_token_player_1}"
P2_TOKEN="${P2_TOKEN:-ganti_token_player_2}"
```

Isi dengan data room yang sedang dites.

Contoh:

```bash
SESSION_ID="${SESSION_ID:-47c8a9e3-22d9-46b1-8f9a-efdc407926d1}"
ROOM_CODE="${ROOM_CODE:-7N4EDX}"
```

Token harus token login terbaru. Jika muncul `invalid or expired token`, login ulang dan ganti token di script.

## 3. Join Player 2 Ke Room

Jika room baru dibuat oleh player 1, player 2 harus join dulu:

```bash
./game-ws-test.sh join-p2
```

Response sukses:

```json
{"session_id":"...","code":"..."}
```

Jika player 2 belum join, saat connect WebSocket akan muncul:

```json
{"error":"kamu tidak ada di room ini"}
```

## 4. Cek State Room

```bash
./game-ws-test.sh state
```

Pastikan `players` berisi 2 user dan `status` masih `waiting`.

Contoh:

```json
{
  "session_id": "47c8a9e3-22d9-46b1-8f9a-efdc407926d1",
  "code": "7N4EDX",
  "status": "waiting",
  "players": [
    {
      "username": "project1",
      "turn_order": 0
    },
    {
      "username": "project2",
      "turn_order": 1
    }
  ]
}
```

## 5. Connect WebSocket Player 1

Buka terminal pertama:

```bash
./game-ws-test.sh ws-p1
```

Jika berhasil, akan muncul event seperti:

```json
{"type":"player_joined","payload":{"user_id":"..."}}
{"type":"room_state","payload":{"status":"waiting", "...":"..."}}
```

Jangan tutup terminal ini. Terminal ini adalah koneksi realtime player 1.

## 6. Connect WebSocket Player 2

Buka terminal kedua:

```bash
./game-ws-test.sh ws-p2
```

Jika berhasil, akan muncul event `player_joined` dan `room_state`.

## 7. Mulai Game

Di terminal WebSocket player 1 atau host, ketik:

```json
{"type":"start_game","payload":{}}
```

Response sukses:

```json
{
  "type": "game_started",
  "payload": {
    "next_player_id": "7a29f8a7-9aa4-44bc-8e15-d6dbdd1da260",
    "next_player_name": "project1",
    "current_turn_idx": 0
  }
}
```

`next_player_id` menentukan player yang harus melakukan `draw_card`.

## 8. Draw Card

Jika `next_player_id` adalah player 1, kirim dari terminal player 1:

```json
{"type":"draw_card","payload":{}}
```

Jika `next_player_id` adalah player 2, kirim dari terminal player 2:

```json
{"type":"draw_card","payload":{}}
```

Response sukses:

```json
{
  "type": "card_drawn",
  "payload": {
    "card_id": "...",
    "type": "truth",
    "content": "Isi kartu",
    "drawn_by_user_id": "..."
  }
}
```

Jika dikirim dari player yang salah, server akan membalas:

```json
{"type":"error","payload":{"error":"bukan giliran kamu"}}
```

## 9. Submit Result

Setelah kartu muncul, player yang sama harus submit hasil.

Selesai:

```json
{"type":"submit_result","payload":{"result":"done"}}
```

Pass:

```json
{"type":"submit_result","payload":{"result":"pass"}}
```

Response sukses:

```json
{"type":"turn_result","payload":{"player_id":"...","card_id":"...","result":"done","cards_left":9}}
{"type":"turn_changed","payload":{"next_player_id":"...","next_player_name":"...","current_turn_idx":1}}
```

Lanjutkan dari player yang ditunjuk oleh `turn_changed.next_player_id`.

## 10. Siklus Bermain

Siklus normal:

```txt
host -> start_game
current player -> draw_card
current player -> submit_result
server -> turn_changed
next player -> draw_card
next player -> submit_result
ulang sampai kartu habis
```

## 11. Stop Game

Host bisa menghentikan game:

```json
{"type":"stop_game","payload":{}}
```

Response:

```json
{"type":"game_finished","payload":{"reason":"host_stopped"}}
```

## Troubleshooting

### `kamu tidak ada di room ini`

Player belum join room. Jalankan:

```bash
./game-ws-test.sh join-p2
```

Lalu coba `ws-p2` lagi.

### `invalid or expired token`

Access token sudah expired atau salah. Login ulang, lalu ganti `P1_TOKEN` atau `P2_TOKEN`.

### `invalid or expired ticket`

Ticket WebSocket sudah expired atau sudah pernah dipakai. Jalankan ulang:

```bash
./game-ws-test.sh ws-p1
```

atau:

```bash
./game-ws-test.sh ws-p2
```

Script akan otomatis mengambil ticket baru.

### Terminal langsung balik ke prompt

Jalankan command lagi dan lihat pesan error. Script sekarang menampilkan response API jika gagal ambil ticket.

Jika ada masalah cache Go:

```bash
GOCACHE=/tmp/go-build ./game-ws-test.sh ws-p1
```

### `minimal 2 pemain untuk memulai`

Player 2 belum join room. Jalankan:

```bash
./game-ws-test.sh join-p2
./game-ws-test.sh state
```

Pastikan ada 2 player.

### `bukan giliran kamu`

Event `draw_card` dikirim dari terminal player yang bukan sedang giliran. Lihat `next_player_id` dari event `game_started` atau `turn_changed`.

## Command Ringkas

```bash
./game-ws-test.sh join-p2
./game-ws-test.sh state
./game-ws-test.sh ws-p1
./game-ws-test.sh ws-p2
```

Event:

```json
{"type":"start_game","payload":{}}
{"type":"draw_card","payload":{}}
{"type":"submit_result","payload":{"result":"done"}}
{"type":"submit_result","payload":{"result":"pass"}}
{"type":"stop_game","payload":{}}
```
