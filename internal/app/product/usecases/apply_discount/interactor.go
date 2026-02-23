package apply_discount

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/bansal247/catalog-proj/internal/app/product/contracts"
	"github.com/bansal247/catalog-proj/internal/app/product/domain"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	"github.com/bansal247/catalog-proj/internal/pkg/committer"
)

type Request struct {
	ProductID          string
	DiscountPercentage float64 // e.g. 20.0 for 20%
	StartDate          time.Time
	EndDate            time.Time
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
		return fmt.Errorf("apply_discount: load: %w", err)
	}
	if product == nil {
		return fmt.Errorf("apply_discount: product %s not found", req.ProductID)
	}

	pctRat := new(big.Rat).SetFloat64(req.DiscountPercentage)
	discount, err := domain.NewDiscount(pctRat, req.StartDate, req.EndDate)
	if err != nil {
		return fmt.Errorf("apply_discount: %w", err)
	}

	if err := product.ApplyDiscount(discount, it.clock.Now()); err != nil {
		return fmt.Errorf("apply_discount: %w", err)
	}

	plan := committer.NewPlan()
	plan.Add(it.productRepo.UpdateMut(product))
	for _, event := range product.DomainEvents() {
		outboxEvent, err := repo.NewOutboxEventFromDomain(event)
		if err != nil {
			return fmt.Errorf("apply_discount: %w", err)
		}
		plan.Add(it.outboxRepo.InsertMut(outboxEvent))
	}

	if err := it.committer.Apply(ctx, plan); err != nil {
		return fmt.Errorf("apply_discount: commit failed: %w", err)
	}
	return nil
}
