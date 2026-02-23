package update_product

import (
	"context"
	"fmt"

	"github.com/bansal247/catalog-proj/internal/app/product/contracts"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	"github.com/bansal247/catalog-proj/internal/pkg/committer"
)

type Request struct {
	ProductID   string
	Name        string
	Description string
	Category    string
}

// Interactor executes the UpdateProduct use case.
type Interactor struct {
	productRepo contracts.ProductRepository
	outboxRepo  contracts.OutboxRepository
	committer   committer.Committer
	clock       clock.Clock
}

// New constructs an UpdateProduct Interactor.
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

// Execute updates a product's mutable details.
func (it *Interactor) Execute(ctx context.Context, req Request) error {
	product, err := it.productRepo.Get(ctx, req.ProductID)
	if err != nil {
		return fmt.Errorf("update_product: load: %w", err)
	}
	if product == nil {
		return fmt.Errorf("update_product: product %s not found", req.ProductID)
	}

	if err := product.Update(req.Name, req.Description, req.Category, it.clock.Now()); err != nil {
		return fmt.Errorf("update_product: %w", err)
	}

	plan := committer.NewPlan()
	plan.Add(it.productRepo.UpdateMut(product))

	for _, event := range product.DomainEvents() {
		outboxEvent, err := repo.NewOutboxEventFromDomain(event)
		if err != nil {
			return fmt.Errorf("update_product: %w", err)
		}
		plan.Add(it.outboxRepo.InsertMut(outboxEvent))
	}

	if err := it.committer.Apply(ctx, plan); err != nil {
		return fmt.Errorf("update_product: commit failed: %w", err)
	}
	return nil
}