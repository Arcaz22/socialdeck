package game

import "github.com/socialdeck/backend/domain"

type Session = domain.Session
type Player = domain.GamePlayer
type RedisSession = domain.RedisSession

type IncomingEvent = domain.IncomingEvent
type OutgoingEvent = domain.OutgoingEvent

type RoomStatePayload = domain.RoomStatePayload
type PlayerInfo = domain.PlayerInfo
type CardDrawnPayload = domain.CardDrawnPayload
type TurnResultPayload = domain.TurnResultPayload
type TurnChangedPayload = domain.TurnChangedPayload
type GameFinishedPayload = domain.GameFinishedPayload

const (
	EventPlayerJoined   = domain.EventPlayerJoined
	EventPlayerLeft     = domain.EventPlayerLeft
	EventGameStarted    = domain.EventGameStarted
	EventCardDrawn      = domain.EventCardDrawn
	EventTurnResult     = domain.EventTurnResult
	EventTurnChanged    = domain.EventTurnChanged
	EventGameFinished   = domain.EventGameFinished
	EventPlayerRejoined = domain.EventPlayerRejoined
	EventError          = domain.EventError
	EventRoomState      = domain.EventRoomState
)
