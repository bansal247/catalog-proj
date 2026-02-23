package domain

import "errors"

var (
	ErrProductNotActive      = errors.New("product is not active")
	ErrProductAlreadyActive  = errors.New("product is already active")
	ErrProductArchived       = errors.New("product is archived")
	ErrInvalidDiscountPeriod = errors.New("discount period is invalid")
	ErrDiscountAlreadyActive = errors.New("product already has an active discount")
	ErrNoActiveDiscount      = errors.New("product has no active discount")
	ErrInvalidPrice          = errors.New("price must be greater than zero")
	ErrInvalidName           = errors.New("product name cannot be empty")
	ErrInvalidCategory       = errors.New("category cannot be empty")
	ErrInvalidDiscount       = errors.New("discount percentage must be between 0 and 100")
	ErrInvalidDenominator    = errors.New("denominator cannot be zero")
)
