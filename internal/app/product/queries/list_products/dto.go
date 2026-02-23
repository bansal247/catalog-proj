package list_products

// ListProductsRequest carries pagination and filter parameters for list queries.
type ListProductsRequest struct {
	Category  string // optional filter
	PageSize  int32
	PageToken string // Spanner resume token (base64 encoded)
}

type ProductList struct {
	Products      []*Product
	NextPageToken string
}

type Product struct {
	ID             string
	Name           string
	Category       string
	EffectivePrice float64
	Status         string
}