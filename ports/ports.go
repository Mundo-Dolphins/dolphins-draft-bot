package ports

import (
	"context"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type ESPNReader interface {
	FetchDraft(ctx context.Context) (domain.DraftSnapshot, error)
}

type TelegramPublisher interface {
	Publish(ctx context.Context, message string) error
}

type EventRepository interface {
	Has(ctx context.Context, key domain.EventKey) (bool, error)
	Save(ctx context.Context, event domain.PublishedEvent) error
	List(ctx context.Context) ([]domain.PublishedEvent, error)
}

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}
