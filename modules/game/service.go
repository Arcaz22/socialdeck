package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	maxPlayers          = 10
	rejoinWindowSeconds = 300 // 5 menit
	wsTicketTTL         = 30 * time.Second
	waitingRoomTTL      = 30 * time.Minute
	activeRoomTTL       = 24 * time.Hour
)

type Service struct {
	repo  *Repository
	redis *redis.Client
}

func NewService(repo *Repository, redis *redis.Client) *Service {
	return &Service{repo: repo, redis: redis}
}

// ─── Room Management ──────────────────────────────────────────────────────────

func (s *Service) CreateRoom(ctx context.Context, deckID, hostID string) (*Session, error) {
	// Pastikan deck ada dan bisa dipakai
	totalCards, err := s.repo.CountDeckCards(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if totalCards == 0 {
		return nil, fmt.Errorf("deck tidak memiliki kartu")
	}

	code, err := generateRoomCode()
	if err != nil {
		return nil, err
	}

	session, err := s.repo.CreateSession(ctx, code, deckID, hostID)
	if err != nil {
		return nil, err
	}

	// Host otomatis join sebagai player pertama
	if _, err := s.repo.AddPlayer(ctx, session.ID, hostID, 0); err != nil {
		return nil, err
	}

	// Init state di Redis
	state := &RedisSession{
		SessionID:      session.ID,
		DeckID:         deckID,
		HostID:         hostID,
		Status:         "waiting",
		PlayerOrder:    []string{hostID},
		CurrentTurnIdx: 0,
		PlayedCardIDs:  []string{},
		TotalCards:     totalCards,
	}
	if err := s.saveRedisState(ctx, session.ID, state); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) JoinRoom(ctx context.Context, code, userID string) (*Session, error) {
	_ = s.expireWaitingRooms(ctx)

	session, err := s.repo.FindSessionByCode(ctx, code)
	if err != nil || session == nil {
		return nil, fmt.Errorf("room tidak ditemukan")
	}
	if session.Status != "waiting" {
		return nil, fmt.Errorf("game sudah dimulai atau selesai")
	}

	count, err := s.repo.CountPlayers(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	// Cek apakah sudah ada di room (rejoin saat waiting)
	exists, _ := s.repo.PlayerExists(ctx, session.ID, userID)
	if exists {
		return session, nil
	}

	if count >= maxPlayers {
		return nil, fmt.Errorf("room sudah penuh (maks %d pemain)", maxPlayers)
	}

	if _, err := s.repo.AddPlayer(ctx, session.ID, userID, count); err != nil {
		return nil, err
	}

	// Update Redis state
	state, err := s.getRedisState(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	state.PlayerOrder = append(state.PlayerOrder, userID)
	if err := s.saveRedisState(ctx, session.ID, state); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *Service) LeaveRoom(ctx context.Context, sessionID, userID string) (bool, error) {
	_ = s.expireWaitingRooms(ctx)

	session, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		return false, err
	}
	if session == nil {
		return false, fmt.Errorf("room tidak ditemukan")
	}
	if session.Status != "waiting" {
		return false, fmt.Errorf("game sudah dimulai, disconnect websocket untuk rejoin")
	}

	state, err := s.getRedisState(ctx, sessionID)
	if err != nil {
		return false, err
	}

	if state.HostID == userID {
		state.Status = "finished"
		s.redis.Del(ctx, s.stateKey(sessionID))
		if err := s.repo.UpdateSessionStatus(ctx, sessionID, "finished"); err != nil {
			return false, err
		}
		return true, nil
	}

	exists, err := s.repo.PlayerExists(ctx, sessionID, userID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, fmt.Errorf("kamu tidak ada di room ini")
	}

	if err := s.repo.RemovePlayer(ctx, sessionID, userID); err != nil {
		return false, err
	}
	state.PlayerOrder = removeUserID(state.PlayerOrder, userID)
	if state.CurrentTurnIdx >= len(state.PlayerOrder) {
		state.CurrentTurnIdx = 0
	}
	if err := s.saveRedisState(ctx, sessionID, state); err != nil {
		return false, err
	}
	s.redis.Del(ctx, s.disconnectKey(sessionID, userID))

	return false, nil
}

func (s *Service) StartGame(ctx context.Context, sessionID, userID string) (*RedisSession, error) {
	_ = s.expireWaitingRooms(ctx)

	state, err := s.getRedisState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state.HostID != userID {
		return nil, fmt.Errorf("hanya host yang bisa memulai game")
	}
	if state.Status != "waiting" {
		return nil, fmt.Errorf("game sudah dimulai")
	}
	if len(state.PlayerOrder) < 2 {
		return nil, fmt.Errorf("minimal 2 pemain untuk memulai")
	}

	state.Status = "active"
	state.CurrentTurnIdx = 0
	if err := s.saveRedisState(ctx, sessionID, state); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSessionStatus(ctx, sessionID, "active"); err != nil {
		return nil, err
	}

	return state, nil
}

// ─── Game Logic ───────────────────────────────────────────────────────────────

func (s *Service) DrawCard(ctx context.Context, sessionID, userID string) (*CardDrawnPayload, error) {
	state, err := s.getRedisState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state.Status != "active" {
		return nil, fmt.Errorf("game belum dimulai")
	}

	// Validasi giliran
	currentPlayerID := state.PlayerOrder[state.CurrentTurnIdx%len(state.PlayerOrder)]
	if currentPlayerID != userID {
		return nil, fmt.Errorf("bukan giliran kamu")
	}

	// Tarik kartu yang belum pernah keluar
	card, err := s.repo.DrawCard(ctx, state.DeckID, state.PlayedCardIDs)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, fmt.Errorf("semua kartu sudah dimainkan")
	}

	// Simpan kartu aktif di Redis sementara
	s.redis.Set(ctx, s.activeCardKey(sessionID), card.ID, 10*time.Minute)

	return &CardDrawnPayload{
		CardID:  card.ID,
		Type:    card.Type,
		Content: card.Content,
		DrawnBy: userID,
	}, nil
}

func (s *Service) SubmitResult(ctx context.Context, sessionID, userID, result string) (*TurnResultPayload, bool, error) {
	if result != "done" && result != "pass" {
		return nil, false, fmt.Errorf("result harus done atau pass")
	}

	state, err := s.getRedisState(ctx, sessionID)
	if err != nil {
		return nil, false, err
	}

	// Ambil kartu aktif dari Redis
	cardID, err := s.redis.Get(ctx, s.activeCardKey(sessionID)).Result()
	if err != nil {
		return nil, false, fmt.Errorf("tidak ada kartu aktif, tarik kartu dulu")
	}

	// Simpan ke DB
	if err := s.repo.RecordPlayedCard(ctx, sessionID, userID, cardID, result); err != nil {
		return nil, false, err
	}

	// Update played cards di Redis
	state.PlayedCardIDs = append(state.PlayedCardIDs, cardID)
	s.redis.Del(ctx, s.activeCardKey(sessionID))

	cardsLeft := state.TotalCards - len(state.PlayedCardIDs)
	gameOver := cardsLeft == 0

	if !gameOver {
		// Pindah giliran
		state.CurrentTurnIdx++
		if err := s.repo.UpdateTurnIdx(ctx, sessionID, state.CurrentTurnIdx); err != nil {
			return nil, false, err
		}
	}

	if err := s.saveRedisState(ctx, sessionID, state); err != nil {
		return nil, false, err
	}

	payload := &TurnResultPayload{
		PlayerID:  userID,
		CardID:    cardID,
		Result:    result,
		CardsLeft: cardsLeft,
	}

	return payload, gameOver, nil
}

func (s *Service) StopGame(ctx context.Context, sessionID, userID string) error {
	state, err := s.getRedisState(ctx, sessionID)
	if err != nil {
		return err
	}
	if state.HostID != userID {
		return fmt.Errorf("hanya host yang bisa menghentikan game")
	}
	return s.finishGame(ctx, sessionID, state)
}

func (s *Service) finishGame(ctx context.Context, sessionID string, state *RedisSession) error {
	state.Status = "finished"
	s.saveRedisState(ctx, sessionID, state)
	return s.repo.UpdateSessionStatus(ctx, sessionID, "finished")
}

// ─── Connection Management ────────────────────────────────────────────────────

func (s *Service) HandleConnect(ctx context.Context, sessionID, userID string) error {
	// Hapus disconnect timer jika ada (rejoin)
	s.redis.Del(ctx, s.disconnectKey(sessionID, userID))
	return s.repo.SetPlayerConnected(ctx, sessionID, userID, true)
}

func (s *Service) HandleDisconnect(ctx context.Context, sessionID, userID string) error {
	s.repo.SetPlayerConnected(ctx, sessionID, userID, false)

	// Set timer 5 menit — jika tidak rejoin, giliran di-skip
	s.redis.Set(ctx,
		s.disconnectKey(sessionID, userID),
		"disconnected",
		time.Duration(rejoinWindowSeconds)*time.Second,
	)
	return nil
}

func (s *Service) IsDisconnected(ctx context.Context, sessionID, userID string) bool {
	err := s.redis.Get(ctx, s.disconnectKey(sessionID, userID)).Err()
	return err == nil // key ada = masih dalam window disconnect
}

func (s *Service) CreateWebSocketTicket(ctx context.Context, sessionID, userID string) (string, error) {
	_ = s.expireWaitingRooms(ctx)

	session, err := s.repo.FindSessionByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session == nil {
		return "", fmt.Errorf("room tidak ditemukan")
	}
	if session.Status == "finished" {
		return "", fmt.Errorf("room sudah selesai")
	}
	if _, err := s.getRedisState(ctx, sessionID); err != nil {
		return "", err
	}

	exists, err := s.repo.PlayerExists(ctx, sessionID, userID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("kamu tidak ada di room ini")
	}

	ticket, err := generateSecureToken(32)
	if err != nil {
		return "", err
	}

	if err := s.redis.Set(ctx, s.wsTicketKey(ticket), sessionID+":"+userID, wsTicketTTL).Err(); err != nil {
		return "", err
	}

	return ticket, nil
}

func (s *Service) ConsumeWebSocketTicket(ctx context.Context, sessionID, ticket string) (string, error) {
	if ticket == "" {
		return "", fmt.Errorf("missing ticket")
	}

	key := s.wsTicketKey(ticket)
	value, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("invalid or expired ticket")
	}
	s.redis.Del(ctx, key)

	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 || parts[0] != sessionID {
		return "", fmt.Errorf("invalid ticket")
	}

	return parts[1], nil
}

// ─── State Helpers ────────────────────────────────────────────────────────────

func (s *Service) GetRedisState(ctx context.Context, sessionID string) (*RedisSession, error) {
	_ = s.expireWaitingRooms(ctx)
	return s.getRedisState(ctx, sessionID)
}

func (s *Service) GetNextPlayerInfo(state *RedisSession, players []Player) *TurnChangedPayload {
	idx := state.CurrentTurnIdx % len(state.PlayerOrder)
	nextID := state.PlayerOrder[idx]

	var nextName string
	for _, p := range players {
		if p.UserID == nextID {
			nextName = p.Username
			break
		}
	}

	return &TurnChangedPayload{
		NextPlayerID:   nextID,
		NextPlayerName: nextName,
		CurrentTurnIdx: state.CurrentTurnIdx,
	}
}

func (s *Service) GetPlayers(ctx context.Context, sessionID string) ([]Player, error) {
	return s.repo.GetPlayers(ctx, sessionID)
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return s.repo.FindSessionByID(ctx, sessionID)
}

// ─── Redis Keys & Helpers ─────────────────────────────────────────────────────

func (s *Service) getRedisState(ctx context.Context, sessionID string) (*RedisSession, error) {
	raw, err := s.redis.Get(ctx, s.stateKey(sessionID)).Bytes()
	if err != nil {
		return nil, fmt.Errorf("session tidak ditemukan di redis")
	}
	var state RedisSession
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Service) saveRedisState(ctx context.Context, sessionID string, state *RedisSession) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}

	ttl := activeRoomTTL
	if state.Status == "waiting" {
		ttl = waitingRoomTTL
	}
	return s.redis.Set(ctx, s.stateKey(sessionID), raw, ttl).Err()
}

func (s *Service) expireWaitingRooms(ctx context.Context) error {
	return s.repo.ExpireWaitingSessions(ctx, waitingRoomTTL)
}

func (s *Service) stateKey(sessionID string) string {
	return fmt.Sprintf("game:state:%s", sessionID)
}

func (s *Service) activeCardKey(sessionID string) string {
	return fmt.Sprintf("game:active_card:%s", sessionID)
}

func (s *Service) disconnectKey(sessionID, userID string) string {
	return fmt.Sprintf("game:disconnect:%s:%s", sessionID, userID)
}

func (s *Service) wsTicketKey(ticket string) string {
	return fmt.Sprintf("game:ws_ticket:%s", ticket)
}

// ─── Room Code Generator ──────────────────────────────────────────────────────

func generateRoomCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // hindari 0,O,1,I (mirip)
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, v := range b {
		sb.WriteByte(chars[int(v)%len(chars)])
	}
	return sb.String(), nil
}

func generateSecureToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func removeUserID(userIDs []string, userID string) []string {
	result := userIDs[:0]
	for _, id := range userIDs {
		if id != userID {
			result = append(result, id)
		}
	}
	return result
}
