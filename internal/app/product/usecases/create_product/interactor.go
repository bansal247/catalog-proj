package create_product

import (
	"context"
	"fmt"

	"github.com/bansal247/catalog-proj/internal/app/product/contracts"
	"github.com/bansal247/catalog-proj/internal/app/product/domain"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	"github.com/bansal247/catalog-proj/internal/pkg/committer"
	"github.com/google/uuid"
)

type Request struct {
	Name        string
	Description string
	Category    string
	BasePriceNumerator   int64
	BasePriceDenominator int64
}

type Interactor struct {
	productRepo contracts.ProductRepository
	outboxRepo  contracts.OutboxRepository
	committer   committer.Committer
	clock       clock.Clock
}

func New(
	productRepo contracts.ProductRepository,
	outboxRepo contracts.OutboxRepository,
	committer committer.Committer,
	clock clock.Clock,
) *Interactor {
	return &Interactor{
		productRepo: productRepo,
		outboxRepo:  outboxRepo,
		committer:   committer,
		clock:       clock,
	}
}

func (it *Interactor) Execute(ctx context.Context, req Request) (string, error) {
	denom := req.BasePriceDenominator
	if denom == 0 {
		denom = 1
	}
	basePrice, err := domain.NewMoney(req.BasePriceNumerator, denom)
	if err != nil {
		return "", fmt.Errorf("create_product: %w", err)
	}

	id := uuid.NewString()
	product, err := domain.NewProduct(id, req.Name, req.Description, req.Category, basePrice, it.clock.Now())
	if err != nil {
		return "", fmt.Errorf("create_product: %w", err)
	}

	plan := committer.NewPlan()
	plan.Add(it.productRepo.InsertMut(product))

	for _, event := range product.DomainEvents() {
		outboxEvent, err := repo.NewOutboxEventFromDomain(event)
		if err != nil {
			return "", fmt.Errorf("create_product: %w", err)
		}
		plan.Add(it.outboxRepo.InsertMut(outboxEvent))
	}

	if err := it.committer.Apply(ctx, plan); err != nil {
		return "", fmt.Errorf("create_product: commit failed: %w", err)
	}

	return product.ID(), nil
}
