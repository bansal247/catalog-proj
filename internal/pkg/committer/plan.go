package committer

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
)


type Mutation any

type Plan struct {
	mutations []Mutation
}

type Committer struct {
	client *spanner.Client
}

func New(client *spanner.Client) *Committer {
	return &Committer{client: client}
}

func (c *Committer) Apply(ctx context.Context, plan *Plan) error {
	if plan.Len() == 0 {
		return nil
	}

	raw := plan.Mutations()
	mutations := make([]*spanner.Mutation, 0, len(raw))

	for i, m := range raw {
		sm, ok := m.(*spanner.Mutation)
		if !ok {
			return fmt.Errorf("spannerdriver: mutation at index %d is %T, expected *spanner.Mutation", i, m)
		}
		mutations = append(mutations, sm)
	}

	_, err := c.client.Apply(ctx, mutations)
	return err
}

func NewPlan() *Plan {
	return &Plan{}
}

func (p *Plan) Add(m Mutation) {
	if m == nil {
		return
	}
	p.mutations = append(p.mutations, m)
}

func (p *Plan) Mutations() []Mutation {
	out := make([]Mutation, len(p.mutations))
	copy(out, p.mutations)
	return out
}

func (p *Plan) Len() int {
	return len(p.mutations)
}

