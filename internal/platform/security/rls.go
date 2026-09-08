package security

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrRLSViolation = errors.New("RLS violation — cross-tenant access blocked")
	ErrTenantNotSet = errors.New("tenant_id not set in session context")
	ErrOrgNotSet    = errors.New("org_id not set in session context")
)

type RLSContext struct {
	TenantID string
	OrgID    string
	UserID   string
}

type RLSManager struct {
	db *sql.DB
}

func NewRLSManager(db *sql.DB) *RLSManager {
	return &RLSManager{db: db}
}

func (m *RLSManager) SetTenantContext(ctx context.Context, rlsCtx RLSContext) error {
	if rlsCtx.TenantID == "" {
		return ErrTenantNotSet
	}

	_, err := m.db.ExecContext(ctx, fmt.Sprintf("SET app.tenant_id = '%s'", rlsCtx.TenantID))
	if err != nil {
		return fmt.Errorf("failed to set tenant_id: %w", err)
	}

	if rlsCtx.OrgID != "" {
		_, err = m.db.ExecContext(ctx, fmt.Sprintf("SET app.org_id = '%s'", rlsCtx.OrgID))
		if err != nil {
			return fmt.Errorf("failed to set org_id: %w", err)
		}
	}

	if rlsCtx.UserID != "" {
		_, err = m.db.ExecContext(ctx, fmt.Sprintf("SET app.user_id = '%s'", rlsCtx.UserID))
		if err != nil {
			return fmt.Errorf("failed to set user_id: %w", err)
		}
	}

	return nil
}

func (m *RLSManager) ClearTenantContext(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "RESET app.tenant_id; RESET app.org_id; RESET app.user_id;")
	return err
}

func (m *RLSManager) VerifyRLSEnabled(ctx context.Context, tableName string) (bool, error) {
	var enabled bool
	query := `
		SELECT coalesce(
			(SELECT relrowsecurity FROM pg_class WHERE relname = $1),
			false
		)`
	err := m.db.QueryRowContext(ctx, query, tableName).Scan(&enabled)
	if err != nil {
		return false, fmt.Errorf("failed to check RLS status: %w", err)
	}
	return enabled, nil
}

func RLSContextFromClaims(claims *Claims) RLSContext {
	return RLSContext{
		TenantID: claims.TenantID,
		OrgID:    claims.OrgID,
		UserID:   claims.UserID,
	}
}
