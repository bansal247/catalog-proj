package m_product

import (
	"time"

	"cloud.google.com/go/spanner"
)

type Data struct {
	ProductID            string
	Name                 string
	Description          string
	Category             string
	BasePriceNumerator   int64
	BasePriceDenominator int64
	DiscountPercent      *float64
	DiscountStartDate    *time.Time
	DiscountEndDate      *time.Time
	Status               string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	ArchivedAt           *time.Time
}

// It is deliberately stateless — create one per repository instance.
type Model struct{}

func New() *Model { return &Model{} }

func (m *Model) InsertMut(d *Data) *spanner.Mutation {
	return spanner.InsertMap(Table, map[string]any{
		ProductID:            d.ProductID,
		Name:                 d.Name,
		Description:          d.Description,
		Category:             d.Category,
		BasePriceNumerator:   d.BasePriceNumerator,
		BasePriceDenominator: d.BasePriceDenominator,
		DiscountPercent:      d.DiscountPercent,
		DiscountStartDate:    d.DiscountStartDate,
		DiscountEndDate:      d.DiscountEndDate,
		Status:               d.Status,
		CreatedAt:            d.CreatedAt,
		UpdatedAt:            d.UpdatedAt,
		ArchivedAt:           d.ArchivedAt,
	})
}

// UpdateMut returns a Spanner UPDATE mutation for the given column map.
// The primary key must be included in cols.
func (m *Model) UpdateMut(productID string, cols map[string]any) *spanner.Mutation {
	cols[ProductID] = productID
	return spanner.UpdateMap(Table, cols)
}
