package evidence_adapter

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/enterprise"
	"github.com/wujupeng/ebcx/internal/evidence"
)

type EnterpriseEvidenceWriter interface {
	Write(ctx context.Context, tx *sql.Tx, agg *enterprise.EnterpriseAggregate, event enterprise.DomainEvent, mutationType string) (*evidence.Record, error)
}
