package repository

import (
	"context"
	"database/sql"

	"github.com/wujupeng/ebcx/internal/platform/security"
)

type UnitOfWork interface {
	BeginTenantTx(ctx context.Context, tenantID string) (*sql.Tx, error)
	Commit(tx *sql.Tx) error
	Rollback(tx *sql.Tx) error
}

type UnitOfWorkPostgreSQL struct {
	rlsManager *security.RLSManager
}

func NewUnitOfWorkPostgreSQL(rlsManager *security.RLSManager) *UnitOfWorkPostgreSQL {
	return &UnitOfWorkPostgreSQL{rlsManager: rlsManager}
}

func (u *UnitOfWorkPostgreSQL) BeginTenantTx(ctx context.Context, tenantID string) (*sql.Tx, error) {
	rlsCtx := security.RLSContext{TenantID: tenantID}
	return u.rlsManager.BeginTenantTransaction(ctx, rlsCtx)
}

func (u *UnitOfWorkPostgreSQL) Commit(tx *sql.Tx) error {
	return tx.Commit()
}

func (u *UnitOfWorkPostgreSQL) Rollback(tx *sql.Tx) error {
	return tx.Rollback()
}
