package get_product

import "time"

// Product is the read-optimised representation returned by queries.
type Product struct {
	ID             string
	Name           string
	Description    string
	Category       string
	BasePrice      float64
	EffectivePrice float64
	Status         string
	Discount       *Discount
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Discount holds discount information for the read model.
type Discount struct {
	Percentage float64
	StartDate  time.Time
	EndDate    time.Time
}