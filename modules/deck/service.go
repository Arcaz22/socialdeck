package deck

import (
	"context"
	"slices"

	"github.com/socialdeck/backend/domain"
)

var validModes = []string{"truth_or_dare", "truth_or_truth", "talk_more"}
var validTypes = []string{"truth", "dare"}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ─── Deck ─────────────────────────────────────────────────────────────────────

func (s *Service) ListPublic(ctx context.Context) ([]domain.Deck, error) {
	return s.repo.ListPublic(ctx)
}

func (s *Service) ListMyDecks(ctx context.Context, userID string) ([]domain.Deck, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *Service) GetDeck(ctx context.Context, deckID, userID string) (*domain.Deck, []domain.Card, error) {
	deck, err := s.repo.FindByID(ctx, deckID)
	if err != nil {
		return nil, nil, err
	}
	if deck == nil {
		return nil, nil, domain.ErrDeckNotFound
	}

	// Akses: deck system/public boleh semua, private hanya owner
	if !deck.IsPublic && !deck.IsSystem {
		if deck.CreatedBy == nil || *deck.CreatedBy != userID {
			return nil, nil, domain.ErrDeckForbidden
		}
	}

	cards, err := s.repo.GetCards(ctx, deckID)
	if err != nil {
		return nil, nil, err
	}

	return deck, cards, nil
}

func (s *Service) CreateDeck(ctx context.Context, name, mode string, isPublic bool, userID string) (*domain.Deck, error) {
	if !slices.Contains(validModes, mode) {
		return nil, domain.ErrInvalidInput("mode tidak valid")
	}
	return s.repo.Create(ctx, name, mode, isPublic, userID)
}

func (s *Service) UpdateDeck(ctx context.Context, deckID, name, mode string, isPublic bool, userID string) error {
	deck, err := s.repo.FindByID(ctx, deckID)
	if err != nil {
		return err
	}
	if deck == nil {
		return domain.ErrDeckNotFound
	}
	if deck.IsSystem {
		return domain.ErrDeckForbidden
	}
	if deck.CreatedBy == nil || *deck.CreatedBy != userID {
		return domain.ErrDeckForbidden
	}
	if !slices.Contains(validModes, mode) {
		return domain.ErrInvalidInput("mode tidak valid")
	}
	return s.repo.Update(ctx, deckID, name, mode, isPublic)
}

func (s *Service) DeleteDeck(ctx context.Context, deckID, userID string) error {
	deck, err := s.repo.FindByID(ctx, deckID)
	if err != nil {
		return err
	}
	if deck == nil {
		return domain.ErrDeckNotFound
	}
	if deck.IsSystem {
		return domain.ErrDeckForbidden
	}
	if deck.CreatedBy == nil || *deck.CreatedBy != userID {
		return domain.ErrDeckForbidden
	}
	return s.repo.Delete(ctx, deckID)
}

// ─── Card ─────────────────────────────────────────────────────────────────────

func (s *Service) AddCard(ctx context.Context, deckID, cardType, content, userID string) (*domain.Card, error) {
	deck, err := s.repo.FindByID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	if deck == nil {
		return nil, domain.ErrDeckNotFound
	}
	if deck.IsSystem {
		return nil, domain.ErrDeckForbidden
	}
	if deck.CreatedBy == nil || *deck.CreatedBy != userID {
		return nil, domain.ErrDeckForbidden
	}
	if !slices.Contains(validTypes, cardType) {
		return nil, domain.ErrInvalidInput("type harus truth atau dare")
	}
	return s.repo.AddCard(ctx, deckID, cardType, content)
}

func (s *Service) DeleteCard(ctx context.Context, deckID, cardID, userID string) error {
	deck, err := s.repo.FindByID(ctx, deckID)
	if err != nil {
		return err
	}
	if deck == nil {
		return domain.ErrDeckNotFound
	}
	if deck.IsSystem {
		return domain.ErrDeckForbidden
	}
	if deck.CreatedBy == nil || *deck.CreatedBy != userID {
		return domain.ErrDeckForbidden
	}
	return s.repo.DeleteCard(ctx, deckID, cardID)
}
