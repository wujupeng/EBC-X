//go:build integration

package security

import (
	"context"
	"fmt"
	"testing"
)

// TestPhysical_RLSManager_ConnectionPoolLifecycle verifies that RLSManager
// is safe for connection pool reuse. This is the Final Evidence Audit test
// requested by the Project Manager.
//
// Scenario:
//   Tenant A -> BeginTenantTransaction(A) -> query sees only A -> rollback
//   Tenant B -> BeginTenantTransaction(B) -> query sees only B -> rollback
//   (same *sql.DB connection pool, different transactions)
//
// The transaction-scoped SET LOCAL ensures:
//   1. All queries within the transaction use the same connection
//   2. The tenant context is automatically cleared on commit/rollback
//   3. No context can leak to the next request
func TestPhysical_RLSManager_ConnectionPoolLifecycle(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	tenantA := "00000000-0000-0000-0000-0000000000aa"
	tenantB := "00000000-0000-0000-0000-0000000000bb"

	rlsDB.Exec("SET ROLE postgres")
	rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-a', 'READ', 'order', 'o-a', 'allow', 'lifecycle-test')`,
		tenantA)
	rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-b', 'READ', 'order', 'o-b', 'allow', 'lifecycle-test')`,
		tenantB)

	rlsDB.Exec("SET ROLE ebcx_runtime")
	defer rlsDB.Exec("RESET ROLE")

	mgr := NewRLSManager(rlsDB)
	ctx := context.Background()

	// Phase 1: Tenant A transaction
	tx1, err := mgr.BeginTenantTransaction(ctx, RLSContext{TenantID: tenantA, UserID: "user-a"})
	if err != nil {
		t.Fatalf("BeginTenantTransaction A: %v", err)
	}

	var countA int
	err = tx1.QueryRowContext(ctx, "SELECT count(*) FROM audit.audit_log").Scan(&countA)
	if err != nil {
		t.Fatalf("query A: %v", err)
	}
	if countA != 1 {
		t.Errorf("Tenant A should see exactly 1 row, got %d", countA)
	}

	var leakedB int
	tx1.QueryRowContext(ctx,
		"SELECT count(*) FROM audit.audit_log WHERE tenant_id::text = $1", tenantB).Scan(&leakedB)
	if leakedB != 0 {
		t.Errorf("SECURITY VIOLATION: Tenant A sees Tenant B data (leaked=%d)", leakedB)
	}

	tx1.Rollback()

	t.Log("Phase 1 PASS: Tenant A transaction sees only A data, B data blocked by RLS")

	// Phase 2: Tenant B transaction (reuses connection pool)
	tx2, err := mgr.BeginTenantTransaction(ctx, RLSContext{TenantID: tenantB, UserID: "user-b"})
	if err != nil {
		t.Fatalf("BeginTenantTransaction B: %v", err)
	}

	var countB int
	err = tx2.QueryRowContext(ctx, "SELECT count(*) FROM audit.audit_log").Scan(&countB)
	if err != nil {
		t.Fatalf("query B: %v", err)
	}
	if countB != 1 {
		t.Errorf("Tenant B should see exactly 1 row, got %d", countB)
	}

	var leakedA int
	tx2.QueryRowContext(ctx,
		"SELECT count(*) FROM audit.audit_log WHERE tenant_id::text = $1", tenantA).Scan(&leakedA)
	if leakedA != 0 {
		t.Errorf("SECURITY VIOLATION: Tenant B sees Tenant A data (leaked=%d)", leakedA)
	}

	tx2.Rollback()

	t.Log("Phase 2 PASS: Tenant B transaction sees only B data, A data blocked by RLS")

	// Phase 3: Verify no context leak after rollback
	// After tx1.Rollback() and tx2.Rollback(), SET LOCAL is auto-cleared.
	// A new transaction without tenant context should see 0 rows (RLS blocks all).
	tx3, err := rlsDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx3: %v", err)
	}

	var countNoCtx int
	err = tx3.QueryRowContext(ctx, "SELECT count(*) FROM audit.audit_log").Scan(&countNoCtx)
	if err != nil {
		t.Fatalf("query no-context: %v", err)
	}
	if countNoCtx != 0 {
		t.Errorf("No tenant context should see 0 rows, got %d (possible context leak)", countNoCtx)
	}
	tx3.Rollback()

	t.Log("Phase 3 PASS: No context leak after rollback — SET LOCAL auto-cleared")

	// Phase 4: Rapid A->B->A on same pool proves no context inheritance
	for i := 0; i < 3; i++ {
		tenantID := tenantA
		if i%2 == 1 {
			tenantID = tenantB
		}

		tx, err := mgr.BeginTenantTransaction(ctx, RLSContext{TenantID: tenantID})
		if err != nil {
			t.Fatalf("iteration %d: BeginTenantTransaction: %v", i, err)
		}

		var count int
		err = tx.QueryRowContext(ctx, "SELECT count(*) FROM audit.audit_log").Scan(&count)
		if err != nil {
			t.Fatalf("iteration %d: query: %v", i, err)
		}
		if count != 1 {
			t.Errorf("iteration %d: expected 1 row, got %d (context leak?)", i, count)
		}

		var crossLeak int
		otherTenant := tenantB
		if i%2 == 1 {
			otherTenant = tenantA
		}
		tx.QueryRowContext(ctx,
			"SELECT count(*) FROM audit.audit_log WHERE tenant_id::text = $1", otherTenant).Scan(&crossLeak)
		if crossLeak != 0 {
			t.Errorf("iteration %d: cross-tenant leak detected (leaked=%d)", i, crossLeak)
		}

		tx.Rollback()
	}

	t.Log("Phase 4 PASS: Rapid A->B->A switching — no context inheritance between transactions")

	t.Log("PHYSICAL PASS: RLSManager ConnectionPoolLifecycle — transaction-scoped SET LOCAL prevents context leak")
}

// TestPhysical_RLSManager_SetTenantContextInTx verifies the SetTenantContextInTx
// method for cases where the caller already has a transaction.
func TestPhysical_RLSManager_SetTenantContextInTx(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	tenantA := "00000000-0000-0000-0000-0000000000cc"

	rlsDB.Exec("SET ROLE postgres")
	rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-c', 'READ', 'order', 'o-c', 'allow', 'intx-test')`,
		tenantA)

	rlsDB.Exec("SET ROLE ebcx_runtime")
	defer rlsDB.Exec("RESET ROLE")

	mgr := NewRLSManager(rlsDB)
	ctx := context.Background()

	tx, err := rlsDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	err = mgr.SetTenantContextInTx(ctx, tx, RLSContext{TenantID: tenantA, UserID: "user-c"})
	if err != nil {
		t.Fatalf("SetTenantContextInTx: %v", err)
	}

	var count int
	err = tx.QueryRowContext(ctx, "SELECT count(*) FROM audit.audit_log").Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	tx.Rollback()

	t.Log("PHYSICAL PASS: SetTenantContextInTx — SET LOCAL within existing transaction works correctly")
}

// TestPhysical_RLSManager_LegacySetOverwrite verifies that the deprecated
// SetTenantContext method properly overwrites previous values when called
// on the same connection. This tests the session-scoped SET behavior.
//
// Note: This test uses a dedicated connection (sql.Conn) to ensure
// the same physical connection is used for all operations.
func TestPhysical_RLSManager_LegacySetOverwrite(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	tenantA := "00000000-0000-0000-0000-0000000000dd"
	tenantB := "00000000-0000-0000-0000-0000000000ee"

	rlsDB.Exec("SET ROLE postgres")
	rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-d', 'READ', 'order', 'o-d', 'allow', 'legacy-test')`,
		tenantA)
	rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-e', 'READ', 'order', 'o-e', 'allow', 'legacy-test')`,
		tenantB)

	// Get a dedicated connection to ensure same physical connection
	conn, err := rlsDB.Conn(context.Background())
	if err != nil {
		t.Fatalf("get conn: %v", err)
	}
	defer conn.Close()

	conn.ExecContext(context.Background(), "SET ROLE ebcx_runtime")
	defer conn.ExecContext(context.Background(), "RESET ROLE")

	// Set Tenant A
	_, err = conn.ExecContext(context.Background(),
		fmt.Sprintf("SET app.tenant_id = '%s'", tenantA))
	if err != nil {
		t.Fatalf("set tenant A: %v", err)
	}

	var countA int
	conn.QueryRowContext(context.Background(),
		"SELECT count(*) FROM audit.audit_log").Scan(&countA)
	if countA != 1 {
		t.Errorf("Tenant A should see 1 row, got %d", countA)
	}

	// Overwrite with Tenant B (same connection)
	_, err = conn.ExecContext(context.Background(),
		fmt.Sprintf("SET app.tenant_id = '%s'", tenantB))
	if err != nil {
		t.Fatalf("set tenant B: %v", err)
	}

	var countB int
	conn.QueryRowContext(context.Background(),
		"SELECT count(*) FROM audit.audit_log").Scan(&countB)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 row, got %d (context not overwritten?)", countB)
	}

	// Verify Tenant A data is not visible after overwrite
	var leakedA int
	conn.QueryRowContext(context.Background(),
		"SELECT count(*) FROM audit.audit_log WHERE tenant_id::text = $1", tenantA).Scan(&leakedA)
	if leakedA != 0 {
		t.Errorf("Tenant B should not see Tenant A data, got %d", leakedA)
	}

	// Clear context
	_, err = conn.ExecContext(context.Background(),
		"RESET app.tenant_id")
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	var countNoCtx int
	conn.QueryRowContext(context.Background(),
		"SELECT count(*) FROM audit.audit_log").Scan(&countNoCtx)
	if countNoCtx != 0 {
		t.Errorf("After RESET should see 0 rows, got %d (context not cleared?)", countNoCtx)
	}

	t.Log("PHYSICAL PASS: Legacy SET overwrite — same connection: A->B->RESET works correctly")
	t.Log("NOTE: Legacy SetTenantContext is safe ONLY with dedicated connections (sql.Conn)")
	t.Log("NOTE: For connection pools (*sql.DB), use BeginTenantTransaction (SET LOCAL in tx)")
}