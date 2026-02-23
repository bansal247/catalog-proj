package domain

import (
	"math/big"
)

type Money struct {
	Amount   *big.Rat
}

func NewMoney(numerator, denominator int64) (*Money, error) {
	if denominator == 0 {
		return nil, ErrInvalidDenominator
	}
	r := big.NewRat(numerator, denominator)
	if r.Sign() <= 0 {
		return nil, ErrInvalidPrice
	}
	return &Money{Amount: r}, nil
}

func NewMoneyFromRat(r *big.Rat) (*Money, error) {
	if r == nil || r.Sign() <= 0 {
		return nil, ErrInvalidPrice
	}
	// Copy to ensure immutability
	copied := new(big.Rat).Set(r)
	return &Money{Amount: copied}, nil
}

func (m *Money) Rat() *big.Rat {return new(big.Rat).Set(m.Amount)}
func (m *Money) Numerator() int64 {return m.Amount.Num().Int64()}
func (m *Money) Denominator() int64 {return m.Amount.Denom().Int64()}
func (m *Money) ApproxString() string {return m.Amount.FloatString(2)}

func (m *Money) ApproxFloat64() float64 {
	f, _ := m.Amount.Float64() // Ignoring error since we know amount is valid and fits in float64
	return f
}

func (m *Money) Equal(other *Money) bool {
	if other == nil {
		return false
	}
	return m.Amount.Cmp(other.Amount) == 0
}

func (m *Money) ApplyPercentageDiscount(discountPercent *big.Rat) (*Money, error) {
	if discountPercent == nil || discountPercent.Sign() < 0 || discountPercent.Cmp(big.NewRat(100, 1)) > 0 {
		return nil, ErrInvalidDiscount
	}
	
	// discount = price * (percent / 100)
	hundredRat := big.NewRat(100, 1)
	discountFraction := new(big.Rat).Quo(discountPercent, hundredRat)
	discountAmount := new(big.Rat).Mul(m.Amount, discountFraction)
	
	finalAmount := new(big.Rat).Sub(m.Amount, discountAmount)

	if finalAmount.Sign() <= 0 {
		return nil, ErrInvalidPrice
	}
	return &Money{Amount: finalAmount}, nil
}