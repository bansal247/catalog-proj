package get_product

import (
	"context"
	"fmt"
)

// Query fetches a single product by ID.
type Query struct {
	readModel Reader
}

// New constructs a GetProduct Query.
func New(readModel Reader) *Query {
	return &Query{readModel: readModel}
}

// Execute returns the product DTO or nil if not found.
func (q *Query) Execute(ctx context.Context, productID string) (*Product, error) {
	dto, err := q.readModel.GetProduct(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get_product: %w", err)
	}
	return dto, nil
}
