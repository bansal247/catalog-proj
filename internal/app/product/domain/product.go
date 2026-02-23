package domain

import "time"

// ProductStatus represents the lifecycle state of a product.
type ProductStatus string
const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	id          string
	name        string
	description string
	category    string
	basePrice   *Money
	discount    *Discount
	status      ProductStatus
	createdAt   time.Time
	updatedAt   time.Time
	archivedAt  *time.Time

	changes *ChangeTracker
	events  []DomainEvent
}

func NewProduct(id, name, description, category string, basePrice *Money, now time.Time) (*Product, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if category == "" {
		return nil, ErrInvalidCategory
	}
	if basePrice == nil {
		return nil, ErrInvalidPrice
	}

	p := &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		status:      ProductStatusDraft,
		createdAt:   now,
		updatedAt:   now,
		changes:     newChangeTracker(),
	}

	p.events = append(p.events, &ProductCreatedEvent{
		ProductID:   id,
		Name:        name,
		Category:    category,
		BasePriceN:  basePrice.Numerator(),
		BasePriceD:  basePrice.Denominator(),
		OccurredAt_: now,
	})

	return p, nil
}

// Reconstitute rebuilds a Product from persisted data without triggering events.
// Used exclusively by the repository layer.
func Reconstitute(
	id, name, description, category string,
	basePrice *Money,
	discount *Discount,
	status ProductStatus,
	createdAt, updatedAt time.Time,
	archivedAt *time.Time,
) *Product {
	return &Product{
		id:          id,
		name:        name,
		description: description,
		category:    category,
		basePrice:   basePrice,
		discount:    discount,
		status:      status,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		archivedAt:  archivedAt,
		changes:     newChangeTracker(),
		events:      nil,
	}
}

// Accessor methods for Product fields. These ensure encapsulation and immutability of internal state.
func (p *Product) ID() string            { return p.id }
func (p *Product) Name() string          { return p.name }
func (p *Product) Description() string   { return p.description }
func (p *Product) Category() string      { return p.category }
func (p *Product) BasePrice() *Money     { return p.basePrice }
func (p *Product) Discount() *Discount   { return p.discount }
func (p *Product) Status() ProductStatus { return p.status }
func (p *Product) CreatedAt() time.Time  { return p.createdAt }
func (p *Product) UpdatedAt() time.Time  { return p.updatedAt }
func (p *Product) ArchivedAt() *time.Time { return p.archivedAt }
func (p *Product) Changes() *ChangeTracker { return p.changes }
func (p *Product) DomainEvents() []DomainEvent { return p.events }

func (p *Product) Update(name, description, category string, now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}
	if name == "" {
		return ErrInvalidName
	}
	if category == "" {
		return ErrInvalidCategory
	}

	p.name = name
	p.description = description
	p.category = category
	p.updatedAt = now
	p.changes.MarkDirty(FieldName)
	p.changes.MarkDirty(FieldDescription)
	p.changes.MarkDirty(FieldCategory)

	p.events = append(p.events, &ProductUpdatedEvent{
		ProductID:   p.id,
		Name:        name,
		Description: description,
		Category:    category,
		OccurredAt_: now,
	})
	return nil
}

func (p *Product) Activate(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}
	if p.status == ProductStatusActive {
		return ErrProductAlreadyActive
	}

	p.status = ProductStatusActive
	p.updatedAt = now
	p.changes.MarkDirty(FieldStatus)

	p.events = append(p.events, &ProductActivatedEvent{
		ProductID:   p.id,
		OccurredAt_: now,
	})
	return nil
}

func (p *Product) Deactivate(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}

	p.status = ProductStatusInactive
	p.updatedAt = now
	p.changes.MarkDirty(FieldStatus)

	p.events = append(p.events, &ProductDeactivatedEvent{
		ProductID:   p.id,
		OccurredAt_: now,
	})
	return nil
}

func (p *Product) Archive(now time.Time) error {
	if p.status == ProductStatusArchived {
		return ErrProductArchived
	}

	p.status = ProductStatusArchived
	p.archivedAt = &now
	p.updatedAt = now
	p.changes.MarkDirty(FieldStatus)
	p.changes.MarkDirty(FieldArchivedAt)

	p.events = append(p.events, &ProductArchivedEvent{
		ProductID:   p.id,
		OccurredAt_: now,
	})

	return nil
}

func (p *Product) ApplyDiscount(discount *Discount, now time.Time) error {
	if p.status != ProductStatusActive {
		return ErrProductNotActive
	}
	if p.discount != nil {
		return ErrDiscountAlreadyActive
	}
	if !discount.IsValidAt(now) {
		return ErrInvalidDiscountPeriod
	}

	p.discount = discount
	p.updatedAt = now
	p.changes.MarkDirty(FieldDiscount)

	p.events = append(p.events, &DiscountAppliedEvent{
		ProductID:           p.id,
		DiscountNumerator:   discount.Numerator(),
		DiscountDenominator: discount.Denominator(),
		StartDate:           discount.StartDate(),
		EndDate:             discount.EndDate(),
		OccurredAt_:         now,
	})
	return nil
}

func (p *Product) RemoveDiscount(now time.Time) error {
	if p.discount == nil {
		return ErrNoActiveDiscount
	}

	p.discount = nil
	p.updatedAt = now
	p.changes.MarkDirty(FieldDiscount)

	p.events = append(p.events, &DiscountRemovedEvent{
		ProductID:   p.id,
		OccurredAt_: now,
	})
	return nil
}