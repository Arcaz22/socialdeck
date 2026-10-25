#!/usr/bin/env bash
set -euo pipefail

# Cara pakai:
# 1. Edit BASE, SESSION_ID, ROOM_CODE, P1_TOKEN, P2_TOKEN di bawah.
# 2. Jalankan:
#      ./game-ws-test.sh join-p2
#      ./game-ws-test.sh state
#      ./game-ws-test.sh ws-p1
#      ./game-ws-test.sh ws-p2
#
# Di terminal WebSocket:
# - Host mulai game:
#      {"type":"start_game","payload":{}}
# - Player yang sedang giliran tarik kartu:
#      {"type":"draw_card","payload":{}}
# - Player yang sama submit hasil:
#      {"type":"submit_result","payload":{"result":"done"}}
#   atau:
#      {"type":"submit_result","payload":{"result":"pass"}}
# - Host stop game:
#      {"type":"stop_game","payload":{}}

BASE="${BASE:-http://localhost:8080/api/v1}"
SESSION_ID="${SESSION_ID:-47c8a9e3-22d9-46b1-8f9a-efdc407926d1}"
ROOM_CODE="${ROOM_CODE:-7N4EDX}"

P1_TOKEN="${P1_TOKEN:-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODY5ODYyMTMsImlhdCI6MTc4Njk4NTMxMywianRpIjoiNmE4MDNmYmEwMGQxZWYwZjAzNGU5NGFiMDNlMTY4MTAiLCJzdWIiOiI3YTI5ZjhhNy05YWE0LTQ0YmMtOGUxNS1kNmRiZGQxZGEyNjAifQ.LoHc-wbG8SVi3Wh0hrjN44VtKKMiVskQHW2KFtCa6_Q}"
P2_TOKEN="${P2_TOKEN:-eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODY5ODYyMzUsImlhdCI6MTc4Njk4NTMzNSwianRpIjoiNmQ4NjFjZDI1YjdiNzM1NDEzOGMxZWI4NTIyM2Q5NDEiLCJzdWIiOiJkOTEzMTNiZi1lMzc4LTRiZGMtODcxOS00MWIxMWY5NmU5MTMifQ.nskeTbfUzItzoDLD7_cHw3dUUbf6qLgpGzdZYmfj3-Y}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Command '$1' belum ada." >&2
    case "$1" in
      jq) echo "Install contoh: sudo apt install jq" >&2 ;;
      go) echo "Install Go dulu, atau jalankan script ini dari environment backend yang sudah bisa go run ." >&2 ;;
    esac
    exit 1
  fi
}

json() {
  if command -v jq >/dev/null 2>&1; then
    jq
  else
    cat
  fi
}

token_for_player() {
  case "$1" in
    p1) printf '%s' "$P1_TOKEN" ;;
    p2) printf '%s' "$P2_TOKEN" ;;
    *) echo "Player harus p1 atau p2" >&2; exit 1 ;;
  esac
}

ensure_token_not_placeholder() {
  local token="$1"
  local name="$2"

  if [[ "$token" == ganti_token_* || -z "$token" ]]; then
    echo "$name belum diisi. Edit file ini atau jalankan dengan env var."
    exit 1
  fi
}

get_ticket() {
  local player="$1"
  local token
  local response
  local ticket

  token="$(token_for_player "$player")"
  ensure_token_not_placeholder "$token" "${player^^}_TOKEN"

  response="$(curl -sS -X POST "$BASE/game/rooms/$SESSION_ID/ws-ticket" \
    -H "Authorization: Bearer $token")"

  ticket="$(printf '%s' "$response" | sed -n 's/.*"ticket"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  if [[ -z "$ticket" ]]; then
    echo "Gagal ambil ticket untuk $player. Response API:" >&2
    echo "$response" >&2
    return 1
  fi

  printf '%s\n' "$ticket"
}

join_p2() {
  ensure_token_not_placeholder "$P2_TOKEN" "P2_TOKEN"

  curl -s -X POST "$BASE/game/rooms/join" \
    -H "Authorization: Bearer $P2_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"$ROOM_CODE\"}" | json
}

state() {
  ensure_token_not_placeholder "$P1_TOKEN" "P1_TOKEN"

  curl -s "$BASE/game/rooms/$SESSION_ID/state" \
    -H "Authorization: Bearer $P1_TOKEN" | json
}

ws_connect() {
  local player="$1"
  local ticket="${2:-}"

  need_cmd go

  if [[ -z "$ticket" ]]; then
    ticket="$(get_ticket "$player")"
  fi

  if [[ -z "$ticket" || "$ticket" == "null" ]]; then
    echo "Gagal ambil ticket untuk $player. Cek token, SESSION_ID, atau status room."
    exit 1
  fi

  echo "Connect sebagai $player ke session $SESSION_ID"
  echo "Ticket sekali pakai dan expired 30 detik."
  GOCACHE="${GOCACHE:-/tmp/go-build}" go run ./cmd/wscli -url "ws://${BASE#http://}/game/rooms/$SESSION_ID/ws?ticket=$ticket"
}

help() {
  cat <<'EOF'
Usage:
  ./game-ws-test.sh join-p2
  ./game-ws-test.sh state
  ./game-ws-test.sh ticket-p1
  ./game-ws-test.sh ticket-p2
  ./game-ws-test.sh ws-p1 [ticket]
  ./game-ws-test.sh ws-p2 [ticket]

Contoh override tanpa edit file:
  P1_TOKEN='eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODY5ODQ4MTUsImlhdCI6MTc4Njk4MzkxNSwianRpIjoiZWI1YWQxZTE5ZDU4NWI4Yzc5MWJlZmQxMTBkZDMzYjciLCJzdWIiOiI3NGFhZDcxYy00ZjM3LTRhZmItYTcyNS0xM2RiYmMxNzk2YWUifQ.dVRKoudrp6XcjAp-kwYy8CDkP9s3s1NU2XGYfLNpSfg' P2_TOKEN='eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODY5ODQ4MjksImlhdCI6MTc4Njk4MzkyOSwianRpIjoiYWU1OWUxMTE3YjZmYTRlYzhmZjFiOWFkYWY4ZmI4MTUiLCJzdWIiOiIyMzEyY2U2MS1kOWM3LTQ5YmQtYmZhZS0yNGZmZjQ3MTlhNmEifQ.O8ZBuNx3h1R_oz_gjKIJwzbrg3ptBBCq6m2CQtml91g' SESSION_ID='314bb32c-4a62-48be-9775-9b4aaf406a70' ROOM_CODE='SE866B' ./game-ws-test.sh state

Event yang diketik di terminal WebSocket:
  {"type":"start_game","payload":{}}
  {"type":"draw_card","payload":{}}
  {"type":"submit_result","payload":{"result":"done"}}
  {"type":"submit_result","payload":{"result":"pass"}}
  {"type":"stop_game","payload":{}}
EOF
}

case "${1:-help}" in
  join-p2) join_p2 ;;
  state) state ;;
  ticket-p1) get_ticket p1 ;;
  ticket-p2) get_ticket p2 ;;
  ws-p1) ws_connect p1 "${2:-}" ;;
  ws-p2) ws_connect p2 "${2:-}" ;;
  help|-h|--help) help ;;
  *) echo "Command tidak dikenal: $1"; help; exit 1 ;;
esac
