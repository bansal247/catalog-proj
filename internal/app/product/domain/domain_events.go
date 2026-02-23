package domain

import "time"

type DomainEvent interface {
	EventType() string
	AggregateID() string
	OccurredAt() time.Time
}


type ProductCreatedEvent struct {
	ProductID   string
	Name        string
	Category    string
	BasePriceN  int64
	BasePriceD  int64
	OccurredAt_ time.Time
}

func (e *ProductCreatedEvent) EventType() string    { return "product.created" }
func (e *ProductCreatedEvent) AggregateID() string  { return e.ProductID }
func (e *ProductCreatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type ProductUpdatedEvent struct {
	ProductID   string
	Name        string
	Description string
	Category    string
	OccurredAt_ time.Time
}

func (e *ProductUpdatedEvent) EventType() string    { return "product.updated" }
func (e *ProductUpdatedEvent) AggregateID() string  { return e.ProductID }
func (e *ProductUpdatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type ProductActivatedEvent struct {
	ProductID   string
	OccurredAt_ time.Time
}

func (e *ProductActivatedEvent) EventType() string    { return "product.activated" }
func (e *ProductActivatedEvent) AggregateID() string  { return e.ProductID }
func (e *ProductActivatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type ProductDeactivatedEvent struct {
	ProductID   string
	OccurredAt_ time.Time
}

func (e *ProductDeactivatedEvent) EventType() string    { return "product.deactivated" }
func (e *ProductDeactivatedEvent) AggregateID() string  { return e.ProductID }
func (e *ProductDeactivatedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type ProductArchivedEvent struct {
	ProductID   string
	OccurredAt_ time.Time
}

func (e *ProductArchivedEvent) EventType() string    { return "product.archived" }
func (e *ProductArchivedEvent) AggregateID() string  { return e.ProductID }
func (e *ProductArchivedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type DiscountAppliedEvent struct {
	ProductID          string
	DiscountNumerator     int64
	DiscountDenominator   int64
	StartDate          time.Time
	EndDate            time.Time
	OccurredAt_        time.Time
}

func (e *DiscountAppliedEvent) EventType() string    { return "product.discount_applied" }
func (e *DiscountAppliedEvent) AggregateID() string  { return e.ProductID }
func (e *DiscountAppliedEvent) OccurredAt() time.Time { return e.OccurredAt_ }

type DiscountRemovedEvent struct {
	ProductID   string
	OccurredAt_ time.Time
}

func (e *DiscountRemovedEvent) EventType() string    { return "product.discount_removed" }
func (e *DiscountRemovedEvent) AggregateID() string  { return e.ProductID }
func (e *DiscountRemovedEvent) OccurredAt() time.Time { return e.OccurredAt_ }


