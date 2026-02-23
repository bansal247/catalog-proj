package product

import (
	"context"

	"github.com/bansal247/catalog-proj/internal/app/product/queries/get_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/activate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/apply_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/create_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/deactivate_product"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/remove_discount"
	"github.com/bansal247/catalog-proj/internal/app/product/usecases/update_product"
	"github.com/bansal247/catalog-proj/internal/transport/grpc/product/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


type Handler struct {
	pb.UnimplementedProductServiceServer

	createProduct    *create_product.Interactor
	updateProduct    *update_product.Interactor
	activateProduct  *activate_product.Interactor
	deactivateProduct *deactivate_product.Interactor
	applyDiscount    *apply_discount.Interactor
	removeDiscount   *remove_discount.Interactor
	getProduct       *get_product.Query
	listProducts     *list_products.Query
}

func New(
	create *create_product.Interactor,
	update *update_product.Interactor,
	activate *activate_product.Interactor,
	deactivate *deactivate_product.Interactor,
	applyDisc *apply_discount.Interactor,
	removeDisc *remove_discount.Interactor,
	getQ *get_product.Query,
	listQ *list_products.Query,
) *Handler {
	return &Handler{
		createProduct:    create,
		updateProduct:    update,
		activateProduct:  activate,
		deactivateProduct: deactivate,
		applyDiscount:    applyDisc,
		removeDiscount:   removeDisc,
		getProduct:       getQ,
		listProducts:     listQ,
	}
}

func (h *Handler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductReply, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	appReq := create_product.Request{
		Name:                 req.Name,
		Description:          req.Description,
		Category:             req.Category,
		BasePriceNumerator:   req.BasePriceNumerator,
		BasePriceDenominator: req.BasePriceDenominator,
	}

	productID, err := h.createProduct.Execute(ctx, appReq)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.CreateProductReply{ProductId: productID}, nil
}

func (h *Handler) UpdateProduct(ctx context.Context, req *pb.UpdateProductRequest) (*pb.UpdateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	appReq := update_product.Request{
		ProductID:   req.ProductId,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
	}

	if err := h.updateProduct.Execute(ctx, appReq); err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.UpdateProductReply{}, nil
}

func (h *Handler) ActivateProduct(ctx context.Context, req *pb.ActivateProductRequest) (*pb.ActivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if err := h.activateProduct.Execute(ctx, activate_product.Request{ProductID: req.ProductId}); err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ActivateProductReply{}, nil
}

func (h *Handler) DeactivateProduct(ctx context.Context, req *pb.DeactivateProductRequest) (*pb.DeactivateProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if err := h.deactivateProduct.Execute(ctx, deactivate_product.Request{ProductID: req.ProductId}); err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.DeactivateProductReply{}, nil
}

func (h *Handler) ApplyDiscount(ctx context.Context, req *pb.ApplyDiscountRequest) (*pb.ApplyDiscountReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.StartDate == nil || req.EndDate == nil {
		return nil, status.Error(codes.InvalidArgument, "start_date and end_date are required")
	}

	appReq := apply_discount.Request{
		ProductID:          req.ProductId,
		DiscountPercentage: req.DiscountPercentage,
		StartDate:          req.StartDate.AsTime(),
		EndDate:            req.EndDate.AsTime(),
	}

	if err := h.applyDiscount.Execute(ctx, appReq); err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.ApplyDiscountReply{}, nil
}

func (h *Handler) RemoveDiscount(ctx context.Context, req *pb.RemoveDiscountRequest) (*pb.RemoveDiscountReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if err := h.removeDiscount.Execute(ctx, remove_discount.Request{ProductID: req.ProductId}); err != nil {
		return nil, mapDomainError(err)
	}
	return &pb.RemoveDiscountReply{}, nil
}

func (h *Handler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductReply, error) {
	if req.ProductId == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	dto, err := h.getProduct.Execute(ctx, req.ProductId)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if dto == nil {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	return &pb.GetProductReply{Product: mapProductToProto(dto)}, nil
}

func (h *Handler) ListProducts(ctx context.Context, req *pb.ListProductsRequest) (*pb.ListProductsReply, error) {
	result, err := h.listProducts.Execute(ctx, list_products.ListProductsRequest{
		Category:  req.Category,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}

	reply := &pb.ListProductsReply{
		NextPageToken: result.NextPageToken,
	}
	for _, dto := range result.Products {
		reply.Products = append(reply.Products, mapProductListToProto(dto))
	}
	return reply, nil
}
