
package evidence_adapter

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/organization"
)

type OrganizationEvidenceWriter interface {
	Write(ctx context.Context, tx *sql.Tx, agg *organization.OrganizationAggregate, event organization.DomainEvent, mutationType string) (*evidence.Record, error)
}

type OrganizationTreeStructuralEvidenceWriter interface {
	WriteStructural(ctx context.Context, tx *sql.Tx, orgID string, tenantID string, event organization.DomainEvent, mutationType string, affectedDescendantIDs []string, levelDelta int) (*evidence.Record, error)
}