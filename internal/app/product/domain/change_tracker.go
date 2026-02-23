package domain

// Field name constants for change tracking.
const (
	FieldName        = "name"
	FieldDescription = "description"
	FieldCategory    = "category"
	FieldBasePrice   = "base_price"
	FieldDiscount    = "discount"
	FieldStatus      = "status"
	FieldArchivedAt  = "archived_at"
)

// ChangeTracker records which fields have been mutated on an aggregate.
// Repositories read this to build targeted, minimal UPDATE mutations.
type ChangeTracker struct {
	dirtyFields map[string]bool
}

func newChangeTracker() *ChangeTracker {
	return &ChangeTracker{dirtyFields: make(map[string]bool)}
}

func (ct *ChangeTracker) MarkDirty(field string) {
	ct.dirtyFields[field] = true
}

// Dirty returns true if the named field has been changed.
func (ct *ChangeTracker) Dirty(field string) bool {
	return ct.dirtyFields[field]
}

// HasChanges returns true if any fields have been marked dirty.
func (ct *ChangeTracker) HasChanges() bool {
	return len(ct.dirtyFields) > 0
}
