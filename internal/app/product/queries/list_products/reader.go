package list_products

import "context"

type Reader interface {
    ListProducts(ctx context.Context, req ListProductsRequest) (*ProductList, error)
}