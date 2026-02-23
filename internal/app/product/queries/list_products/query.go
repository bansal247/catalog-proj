package list_products

import (
	"context"
	"fmt"
)

// Query lists products with optional filtering and pagination.
type Query struct {
	readModel Reader
}

// New constructs a ListProducts Query.
func New(readModel Reader) *Query {
	return &Query{readModel: readModel}
}

// Execute returns a page of products.
func (q *Query) Execute(ctx context.Context, req ListProductsRequest) (*ProductList, error) {
	result, err := q.readModel.ListProducts(ctx, ListProductsRequest{
		Category:  req.Category,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	})
	if err != nil {
		return nil, fmt.Errorf("list_products: %w", err)
	}
	return result, nil
}