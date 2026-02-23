package services

import (
	"time"

	"github.com/bansal247/catalog-proj/internal/app/product/domain"
)

type PricingCalculator struct{}

func NewPricingCalculator() *PricingCalculator {
	return &PricingCalculator{}
}


func (c *PricingCalculator) EffectivePrice(p *domain.Product, t time.Time) (*domain.Money, error) {
	d := p.Discount()
	if d == nil || !d.IsValidAt(t) {
		return p.BasePrice(), nil
	}

	discounted, err := p.BasePrice().ApplyPercentageDiscount(d.Percentage())
	if err != nil {
		return nil, err
	}
	return discounted, nil
}