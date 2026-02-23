package m_product

// Column name constants for the products Spanner table.
// Using constants prevents typos and makes field references grep-able.
const (
	Table = "products"

	ProductID            = "product_id"
	Name                 = "name"
	Description          = "description"
	Category             = "category"
	BasePriceNumerator   = "base_price_numerator"
	BasePriceDenominator = "base_price_denominator"
	DiscountPercent      = "discount_percent"
	DiscountStartDate    = "discount_start_date"
	DiscountEndDate      = "discount_end_date"
	Status               = "status"
	CreatedAt            = "created_at"
	UpdatedAt            = "updated_at"
	ArchivedAt           = "archived_at"
)

// AllColumns lists every column in table read order (used for SELECT *).
var AllColumns = []string{
	ProductID,
	Name,
	Description,
	Category,
	BasePriceNumerator,
	BasePriceDenominator,
	DiscountPercent,
	DiscountStartDate,
	DiscountEndDate,
	Status,
	CreatedAt,
	UpdatedAt,
	ArchivedAt,
}
