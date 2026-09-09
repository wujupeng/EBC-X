package repository

import (
	"context"
	"database/sql"
)

type CommandFirstResult struct {
	ResultVersion    int64
	ResultEventID    string
	ResultEvidenceID string
}

type CommandIdempotencyRepository interface {
	CheckAndReserve(ctx context.Context, tx *sql.Tx, commandID, tenantID, commandType, aggregateID string) (*CommandFirstResult, error)
	MarkSuccess(ctx context.Context, tx *sql.Tx, commandID, tenantID string, resultVersion int64, resultEventID, resultEvidenceID string) error
	MarkFailed(ctx context.Context, tx *sql.Tx, commandID, tenantID string) error
}
