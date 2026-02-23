package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"

	"github.com/bansal247/catalog-proj/internal/app/product/contracts"
	"github.com/bansal247/catalog-proj/internal/app/product/domain"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/get_product"
	"github.com/bansal247/catalog-proj/internal/app/product/queries/list_products"
	"github.com/bansal247/catalog-proj/internal/models/m_outbox"
	"github.com/bansal247/catalog-proj/internal/models/m_product"
)

type pricingService interface {
	EffectivePrice(p *domain.Product, t time.Time) (*domain.Money, error)
}

type ProductRepo struct {
	client *spanner.Client
	model  *m_product.Model
}

type SpannerReadModel struct {
	client    *spanner.Client
	priceSvc  pricingService
}

// OutboxRepo stores domain events in the transactional outbox table.
type OutboxRepo struct {
	model *m_outbox.Model
}

func NewProductRepo(client *spanner.Client) *ProductRepo {
	return &ProductRepo{
		client: client,
		model:  m_product.New(),
	}
}

func (r *ProductRepo) Get(ctx context.Context, id string) (*domain.Product, error) {
	row, err := r.client.Single().ReadRow(ctx, m_product.Table, spanner.Key{id}, m_product.AllColumns)
	if err != nil {
		return nil, fmt.Errorf("productRepo.Get: %w", err)
	}

	return r.rowToProduct(row)
}

// InsertMut returns a Spanner INSERT mutation for a freshly-created product.
func (r *ProductRepo) InsertMut(p *domain.Product) *spanner.Mutation {
	d := r.productToData(p)
	return r.model.InsertMut(d)
}

// UpdateMut inspects the ChangeTracker and builds a targeted UPDATE mutation.
// Returns nil when no fields are dirty.
func (r *ProductRepo) UpdateMut(p *domain.Product) *spanner.Mutation {
	cols := make(map[string]interface{})

	if p.Changes().Dirty(domain.FieldName) {
		cols[m_product.Name] = p.Name()
	}
	if p.Changes().Dirty(domain.FieldDescription) {
		cols[m_product.Description] = p.Description()
	}
	if p.Changes().Dirty(domain.FieldCategory) {
		cols[m_product.Category] = p.Category()
	}
	if p.Changes().Dirty(domain.FieldStatus) {
		cols[m_product.Status] = string(p.Status())
	}
	if p.Changes().Dirty(domain.FieldArchivedAt) {
		cols[m_product.ArchivedAt] = p.ArchivedAt()
	}
	if p.Changes().Dirty(domain.FieldDiscount) {
		if d := p.Discount(); d != nil {
			pct := d.PercentageFloat64()
			start := d.StartDate()
			end := d.EndDate()
			cols[m_product.DiscountPercent] = pct
			cols[m_product.DiscountStartDate] = start
			cols[m_product.DiscountEndDate] = end
		} else {
			cols[m_product.DiscountPercent] = nil
			cols[m_product.DiscountStartDate] = nil
			cols[m_product.DiscountEndDate] = nil
		}
	}

	if len(cols) == 0 {
		return nil
	}
	cols[m_product.UpdatedAt] = time.Now()
	return r.model.UpdateMut(p.ID(), cols)
}

func NewSpannerReadModel(client *spanner.Client, priceSvc pricingService) *SpannerReadModel {
	return &SpannerReadModel{client: client, priceSvc: priceSvc}
}

// GetProduct loads a single product and computes its effective price.
func (rm *SpannerReadModel) GetProduct(ctx context.Context, id string) (*get_product.Product, error) {
	repo := NewProductRepo(rm.client)
	p, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	
	return rm.toGetProductDTO(p)
}

// ListProducts returns a paginated list of non-archived products.
func (rm *SpannerReadModel) ListProducts(ctx context.Context, req list_products.ListProductsRequest) (*list_products.ProductList, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var stmt spanner.Statement
	if req.Category != "" {
		stmt = spanner.Statement{
			SQL: `SELECT ` + joinCols() + ` FROM products
			      WHERE status != 'archived' AND category = @category
			      ORDER BY created_at DESC
			      LIMIT @limit`,
			Params: map[string]interface{}{
				"category": req.Category,
				"limit":    int64(pageSize + 1),
			},
		}
	} else {
		stmt = spanner.Statement{
			SQL: `SELECT ` + joinCols() + ` FROM products
			      WHERE status != 'archived'
			      ORDER BY created_at DESC
			      LIMIT @limit`,
			Params: map[string]interface{}{
				"limit": int64(pageSize + 1),
			},
		}
	}

	iter := rm.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var products []*list_products.Product
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listProducts: %w", err)
		}
		p, err := rowToProduct(row)
		if err != nil {
			return nil, err
		}
		dto, err := rm.toListProductDTO(p)
		if err != nil {
			return nil, err
		}
		products = append(products, dto)
	}

	var nextToken string
	if len(products) > int(pageSize) {
		products = products[:pageSize]
		nextToken = "more" // simplified — production would use a real cursor
	}

	return &list_products.ProductList{
		Products:      products,
		NextPageToken: nextToken,
	}, nil
}

func (rm *SpannerReadModel) toGetProductDTO(p *domain.Product) (*get_product.Product, error) {
	effective, err := rm.priceSvc.EffectivePrice(p, time.Now())
	if err != nil {
		return nil, err
	}

	dto := &get_product.Product{
		ID:             p.ID(),
		Name:           p.Name(),
		Description:    p.Description(),
		Category:       p.Category(),
		BasePrice:      p.BasePrice().ApproxFloat64(),
		EffectivePrice: effective.ApproxFloat64(),
		Status:         string(p.Status()),
		CreatedAt:      p.CreatedAt(),
		UpdatedAt:      p.UpdatedAt(),
	}

	if d := p.Discount(); d != nil {
		dto.Discount = &get_product.Discount{
			Percentage: d.PercentageFloat64(),
			StartDate:  d.StartDate(),
			EndDate:    d.EndDate(),
		}
	}
	return dto, nil
}

func (rm *SpannerReadModel) toListProductDTO(p *domain.Product) (*list_products.Product, error) {
	effective, err := rm.priceSvc.EffectivePrice(p, time.Now())
	if err != nil {
		return nil, err
	}

	dto := &list_products.Product{
		ID:             p.ID(),
		Name:           p.Name(),
		Category:       p.Category(),
		EffectivePrice: effective.ApproxFloat64(),
		Status:         string(p.Status()),
	}

	return dto, nil
}

func NewOutboxRepo() *OutboxRepo {
	return &OutboxRepo{model: m_outbox.New()}
}

// InsertMut returns the mutation for persisting a new outbox event.
func (r *OutboxRepo) InsertMut(event *contracts.OutboxEvent) *spanner.Mutation {
	d := &m_outbox.Data{
		EventID:     event.ID,
		EventType:   event.EventType,
		AggregateID: event.AggregateID,
		Payload:     event.Payload,
		Status:      m_outbox.StatusPending,
		CreatedAt:   time.Now(),
	}
	return r.model.InsertMut(d)
}

//helpers

func (r *ProductRepo) productToData(p *domain.Product) *m_product.Data {
	d := &m_product.Data{
		ProductID:            p.ID(),
		Name:                 p.Name(),
		Description:          p.Description(),
		Category:             p.Category(),
		BasePriceNumerator:   p.BasePrice().Numerator(),
		BasePriceDenominator: p.BasePrice().Denominator(),
		Status:               string(p.Status()),
		CreatedAt:            p.CreatedAt(),
		UpdatedAt:            p.UpdatedAt(),
		ArchivedAt:           p.ArchivedAt(),
	}
	if disc := p.Discount(); disc != nil {
		pct := disc.PercentageFloat64()
		start := disc.StartDate()
		end := disc.EndDate()
		d.DiscountPercent = &pct
		d.DiscountStartDate = &start
		d.DiscountEndDate = &end
	}
	return d
}

func (r *ProductRepo) rowToProduct(row *spanner.Row) (*domain.Product, error) {
	return rowToProduct(row)
}

func rowToProduct(row *spanner.Row) (*domain.Product, error) {
	var (
		id, name, description, category, status string
		basePriceN, basePriceD                  int64
		discountPct                              *float64
		discountStart, discountEnd               *time.Time
		createdAt, updatedAt                     time.Time
		archivedAt                               *time.Time
	)

	if err := row.Columns(
		&id, &name, &description, &category,
		&basePriceN, &basePriceD,
		&discountPct, &discountStart, &discountEnd,
		&status, &createdAt, &updatedAt, &archivedAt,
	); err != nil {
		return nil, fmt.Errorf("rowToProduct: %w", err)
	}

	basePrice, err := domain.NewMoney(basePriceN, basePriceD)
	if err != nil {
		return nil, fmt.Errorf("rowToProduct: invalid base price: %w", err)
	}

	var discount *domain.Discount
	if discountPct != nil && discountStart != nil && discountEnd != nil {
		pctRat := new(big.Rat).SetFloat64(*discountPct)
		discount, err = domain.NewDiscount(pctRat, *discountStart, *discountEnd)
		if err != nil {
			return nil, fmt.Errorf("rowToProduct: invalid discount: %w", err)
		}
	}

	return domain.Reconstitute(
		id, name, description, category,
		basePrice, discount,
		domain.ProductStatus(status),
		createdAt, updatedAt, archivedAt,
	), nil
}

func joinCols() string {
	cols := m_product.AllColumns
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}

func NewOutboxEventFromDomain(event domain.DomainEvent) (*contracts.OutboxEvent, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal domain event: %w", err)
	}
	return &contracts.OutboxEvent{
		ID:          uuid.NewString(),
		EventType:   event.EventType(),
		AggregateID: event.AggregateID(),
		Payload:     payload,
	}, nil
}
