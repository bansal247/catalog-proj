package product

import (
	"errors"

	"github.com/bansal247/catalog-proj/internal/app/product/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mapDomainError converts domain sentinel errors and other known errors to
// appropriate gRPC status codes. Unknown errors become Internal.
func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrProductNotActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrProductAlreadyActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrProductArchived):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidDiscountPeriod):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrDiscountAlreadyActive):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrNoActiveDiscount):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidCategory):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidDiscount):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Errorf(codes.Internal, "internal error: %v", err)
	}
}

// validateCreateRequest performs structural validation on a CreateProductRequest.
func validateCreateRequest(req interface {
	GetName() string
	GetCategory() string
	GetBasePriceNumerator() int64
	GetBasePriceDenominator() int64
}) error {
	if req.GetName() == "" {
		return errors.New("name is required")
	}
	if req.GetCategory() == "" {
		return errors.New("category is required")
	}
	if req.GetBasePriceNumerator() <= 0 {
		return errors.New("base_price_numerator must be > 0")
	}
	denom := req.GetBasePriceDenominator()
	if denom == 0 {
		denom = 1
	}
	if denom < 0 {
		return errors.New("base_price_denominator must be > 0")
	}
	return nil
}