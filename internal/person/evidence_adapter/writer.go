package evidence_adapter

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/person"
)

type PersonEvidenceWriter interface {
	Write(ctx context.Context, tx *sql.Tx, agg *person.PersonAggregate, event person.DomainEvent, mutationType string) (*evidence.Record, error)
}
