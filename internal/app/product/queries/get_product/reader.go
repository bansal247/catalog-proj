package get_product

import "context"

type Reader interface {
    GetProduct(ctx context.Context, id string) (*Product, error)
}