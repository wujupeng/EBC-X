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

// SetTenantContext sets tenant context using session-scoped SET.
//
// Deprecated: This method is NOT safe for connection pools (*sql.DB).
// SET only affects one connection in the pool; subsequent queries may
// use a different connection without the tenant context. Use
// BeginTenantTransaction or SetTenantContextInTx instead.
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

// ClearTenantContext resets all RLS session variables.
//
// Deprecated: Not safe for connection pools. Use transaction-scoped
// methods (BeginTenantTransaction) which auto-clear on commit/rollback.
func (m *RLSManager) ClearTenantContext(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, "RESET app.tenant_id; RESET app.org_id; RESET app.user_id;")
	return err
}

// BeginTenantTransaction starts a transaction and sets the tenant context
// using SET LOCAL (transaction-scoped). The context is automatically cleared
// when the transaction commits or rolls back. This is the connection-pool-safe
// method: all queries within the transaction use the same connection, and
// the tenant context cannot leak to subsequent requests.
func (m *RLSManager) BeginTenantTransaction(ctx context.Context, rlsCtx RLSContext) (*sql.Tx, error) {
	if rlsCtx.TenantID == "" {
		return nil, ErrTenantNotSet
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", rlsCtx.TenantID)); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to set tenant_id: %w", err)
	}

	if rlsCtx.OrgID != "" {
		if _, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.org_id = '%s'", rlsCtx.OrgID)); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to set org_id: %w", err)
		}
	}

	if rlsCtx.UserID != "" {
		if _, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.user_id = '%s'", rlsCtx.UserID)); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to set user_id: %w", err)
		}
	}

	return tx, nil
}

// SetTenantContextInTx sets tenant context within an existing transaction
// using SET LOCAL (transaction-scoped). The context is automatically cleared
// when the transaction commits or rolls back. Use this when you already have
// a transaction and need to set the RLS context within it.
func (m *RLSManager) SetTenantContextInTx(ctx context.Context, tx *sql.Tx, rlsCtx RLSContext) error {
	if rlsCtx.TenantID == "" {
		return ErrTenantNotSet
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", rlsCtx.TenantID)); err != nil {
		return fmt.Errorf("failed to set tenant_id: %w", err)
	}

	if rlsCtx.OrgID != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.org_id = '%s'", rlsCtx.OrgID)); err != nil {
			return fmt.Errorf("failed to set org_id: %w", err)
		}
	}

	if rlsCtx.UserID != "" {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.user_id = '%s'", rlsCtx.UserID)); err != nil {
			return fmt.Errorf("failed to set user_id: %w", err)
		}
	}

	return nil
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
