package product

import (
	"github.com/bansal247/catalog-proj/internal/app/product/queries/get_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
	"github.com/bansal247/catalog-proj/internal/transport/grpc/product/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func mapProductToProto(dto *get_product.Product) *pb.ProductProto {
	p := &pb.ProductProto{
		ProductId:      dto.ID,
		Name:           dto.Name,
		Description:    dto.Description,
		Category:       dto.Category,
		BasePrice:      dto.BasePrice,
		EffectivePrice: dto.EffectivePrice,
		Status:         dto.Status,
		CreatedAt:      timestamppb.New(dto.CreatedAt),
		UpdatedAt:      timestamppb.New(dto.UpdatedAt),
	}

	if dto.Discount != nil {
		p.Discount = &pb.DiscountProto{
			Percentage: dto.Discount.Percentage,
			StartDate:  timestamppb.New(dto.Discount.StartDate),
			EndDate:    timestamppb.New(dto.Discount.EndDate),
		}
	}
	return p
}

func mapProductListToProto(dto *list_products.Product) *pb.ProductProto {
	p := &pb.ProductProto{
		ProductId:      dto.ID,
		Name:           dto.Name,
		Category:       dto.Category,
		EffectivePrice: dto.EffectivePrice,
		Status:         dto.Status,
	}
	return p
}