package contracts

import (
	"context"

	"cloud.google.com/go/spanner"

	"github.com/bansal247/catalog-proj/internal/app/product/domain"
)

// ProductRepository defines the write-side data access contract.
// Methods return Spanner mutations but never apply them — the usecase
// is responsible for collecting and committing mutations via CommitPlan.
type ProductRepository interface {
	Get(ctx context.Context, id string) (*domain.Product, error)
	InsertMut(p *domain.Product) *spanner.Mutation
	UpdateMut(p *domain.Product) *spanner.Mutation
}

// OutboxRepository persists domain events in the transactional outbox table.
type OutboxRepository interface {
	InsertMut(event *OutboxEvent) *spanner.Mutation
}

type OutboxEvent struct {
	ID          string
	EventType   string
	AggregateID string
	Payload     []byte
}