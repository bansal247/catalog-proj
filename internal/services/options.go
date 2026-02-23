// Package services wires all application dependencies together.
package services

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"github.com/bansal247/catalog-proj/internal/app/product/domain/services"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/get_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
	"github.com/bansal247/catalog-proj/internal/app/product/repo"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/activate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/apply_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/create_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/deactivate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/remove_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/update_product"
	"github.com/bansal247/catalog-proj/internal/pkg/clock"
	spannerdriver "github.com/bansal247/catalog-proj/internal/pkg/committer"
	grpchandler "github.com/bansal247/catalog-proj/internal/transport/grpc/product"
)

type Options struct {
	SpannerClient *spanner.Client

	// Commands
	CreateProduct    *create_product.Interactor
	UpdateProduct    *update_product.Interactor
	ActivateProduct  *activate_product.Interactor
	DeactivateProduct *deactivate_product.Interactor
	ApplyDiscount    *apply_discount.Interactor
	RemoveDiscount   *remove_discount.Interactor

	// Queries
	GetProduct   *get_product.Query
	ListProducts *list_products.Query

	// gRPC handler
	ProductHandler *grpchandler.Handler
}

// Config holds infrastructure configuration.
type Config struct {
	SpannerProject  string
	SpannerInstance string
	SpannerDatabase string
}

func Build(ctx context.Context, cfg Config) (*Options, error) {
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		cfg.SpannerProject, cfg.SpannerInstance, cfg.SpannerDatabase)

	client, err := spanner.NewClient(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("spanner client: %w", err)
	}
	log.Printf("Connected to Spanner: %s", dbPath)

	return BuildWithClient(client), nil
}

// BuildWithClient wires all services using an already-constructed Spanner client.
// Useful in tests where the client is created against the emulator.
func BuildWithClient(client *spanner.Client) *Options {
	clk := clock.Real{}
	committer := spannerdriver.New(client)
	pricingSvc := services.NewPricingCalculator()

	productRepo := repo.NewProductRepo(client)
	outboxRepo := repo.NewOutboxRepo()
	readModel := repo.NewSpannerReadModel(client, pricingSvc)

	createUC := create_product.New(productRepo, outboxRepo, *committer, clk)
	updateUC := update_product.New(productRepo, outboxRepo, *committer, clk)
	activateUC := activate_product.New(productRepo, outboxRepo, *committer, clk)
	deactivateUC := deactivate_product.New(productRepo, outboxRepo, *committer, clk)
	applyDiscUC := apply_discount.New(productRepo, outboxRepo, *committer, clk)
	removeDiscUC := remove_discount.New(productRepo, outboxRepo, *committer, clk)

	getQ := get_product.New(readModel)
	listQ := list_products.New(readModel)

	handler := grpchandler.New(
		createUC, updateUC, activateUC, deactivateUC,
		applyDiscUC, removeDiscUC, getQ, listQ,
	)

	return &Options{
		SpannerClient:    client,
		CreateProduct:    createUC,
		UpdateProduct:    updateUC,
		ActivateProduct:  activateUC,
		DeactivateProduct: deactivateUC,
		ApplyDiscount:    applyDiscUC,
		RemoveDiscount:   removeDiscUC,
		GetProduct:       getQ,
		ListProducts:     listQ,
		ProductHandler:   handler,
	}
}
