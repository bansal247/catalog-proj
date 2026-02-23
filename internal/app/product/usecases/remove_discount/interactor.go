package remove_discount

import (
	"context"
	"fmt"

	"github.com/bansal247/catalog-proj/internal/app/product/contracts"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	"github.com/bansal247/catalog-proj/internal/pkg/committer"
)

type Request struct {
	ProductID string
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
	return &Interactor{productRepo: productRepo, outboxRepo: outboxRepo, committer: committer, clock: clock}
}


func (it *Interactor) Execute(ctx context.Context, req Request) error {
	product, err := it.productRepo.Get(ctx, req.ProductID)
	if err != nil {
		return fmt.Errorf("remove_discount: load: %w", err)
	}
	if product == nil {
		return fmt.Errorf("remove_discount: product %s not found", req.ProductID)
	}

	if err := product.RemoveDiscount(it.clock.Now()); err != nil {
		return fmt.Errorf("remove_discount: %w", err)
	}

	plan := committer.NewPlan()
	plan.Add(it.productRepo.UpdateMut(product))
	for _, event := range product.DomainEvents() {
		outboxEvent, err := repo.NewOutboxEventFromDomain(event)
		if err != nil {
			return fmt.Errorf("remove_discount: %w", err)
		}
		plan.Add(it.outboxRepo.InsertMut(outboxEvent))
	}

	if err := it.committer.Apply(ctx, plan); err != nil {
		return fmt.Errorf("remove_discount: commit failed: %w", err)
	}
	return nil
}