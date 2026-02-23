package m_outbox

import (
	"encoding/json"
	"time"

	"cloud.google.com/go/spanner"
)

type Data struct {
	EventID     string
	EventType   string
	AggregateID string
	Payload     json.RawMessage
	Status      string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

type Model struct{}

func New() *Model { return &Model{} }

// InsertMut returns a Spanner INSERT mutation for a new outbox event.
func (m *Model) InsertMut(d *Data) *spanner.Mutation {
	return spanner.InsertMap(Table, map[string]interface{}{
		EventID:     d.EventID,
		EventType:   d.EventType,
		AggregateID: d.AggregateID,
		Payload:     string(d.Payload),
		Status:      d.Status,
		CreatedAt:   d.CreatedAt,
		ProcessedAt: d.ProcessedAt,
	})
}