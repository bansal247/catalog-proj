package domain

import (
	"math/big"
	"time"
)

// Discount is an immutable value object representing a percentage-based discount
// with a validity window.
type Discount struct {
	percentage *big.Rat
	startDate  time.Time
	endDate    time.Time
}

func NewDiscount(percentage *big.Rat, startDate, endDate time.Time) (*Discount, error) {
	zero := big.NewRat(0, 1)
	hundred := big.NewRat(100, 1)

	if percentage.Cmp(zero) <= 0 || percentage.Cmp(hundred) > 0 {
		return nil, ErrInvalidDiscount
	}

	if !endDate.After(startDate) {
		return nil, ErrInvalidDiscountPeriod
	}

	return &Discount{
		percentage: new(big.Rat).Set(percentage),
		startDate:  startDate,
		endDate:    endDate,
	}, nil
}

// returns a copy of the discount percentage as a *big.Rat.
func (d *Discount) Percentage() *big.Rat {return new(big.Rat).Set(d.percentage)}

func (d *Discount) PercentageFloat64() float64 {
	f, _ := d.percentage.Float64() // Ignoring error since we know percentage is valid and fits in float64
	return f
}
func (d *Discount) StartDate() time.Time {return d.startDate}
func (d *Discount) EndDate() time.Time {return d.endDate}
func (d *Discount) Numerator() int64 {return d.percentage.Num().Int64()}
func (d *Discount) Denominator() int64 {return d.percentage.Denom().Int64()}


func (d *Discount) IsValidAt(t time.Time) bool {
	return !t.Before(d.startDate) && t.Before(d.endDate)
}