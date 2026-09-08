//go:build integration

package physicalgate

import (
	"testing"
)

func TestPhysical007_Audit_RLS_TenantIsolation(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	execShouldSucceed(t, db, "insert audit A",
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason, ip_address)
		 VALUES ($1, 'user-a', 'READ', 'order', 'order-1', 'allow', 'policy-match', '10.0.0.1')`,
		tenantA)

	execShouldSucceed(t, db, "insert audit B",
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason, ip_address)
		 VALUES ($1, 'user-b', 'READ', 'order', 'order-2', 'allow', 'policy-match', '10.0.0.2')`,
		tenantB)

	execShouldSucceed(t, db, "insert audit A second",
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason, ip_address)
		 VALUES ($1, 'user-a', 'WRITE', 'order', 'order-3', 'deny', 'no-permission', '10.0.0.1')`,
		tenantA)

	countA := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM audit.audit_log`)
	if countA != 2 {
		t.Errorf("Tenant A should see 2 audit entries, got %d", countA)
	}

	countB := countRowsAsTenant(t, db, tenantB, `SELECT count(*) FROM audit.audit_log`)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 audit entry, got %d", countB)
	}

	countNone := countRowsAsTenant(t, db, "00000000-0000-0000-0000-000000000099", `SELECT count(*) FROM audit.audit_log`)
	if countNone != 0 {
		t.Errorf("Unknown tenant should see 0 audit entries, got %d", countNone)
	}

	t.Log("PHYSICAL PASS: Audit RLS -?Tenant A sees 2, Tenant B sees 1, unknown tenant sees 0")
}

func TestPhysical007_Audit_WOM_Triggers(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"

	execShouldSucceed(t, db, "insert audit for WORM test",
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason, ip_address)
		 VALUES ($1, 'user-w', 'DELETE', 'artifact', 'art-1', 'deny', 'worm-protected', '10.0.0.9')`,
		tenantA)

	execShouldFail(t, db, "UPDATE audit.audit_log (append-only violation)",
		`UPDATE audit.audit_log SET decision = 'allow' WHERE resource = 'artifact' AND resource_id = 'art-1'`)

	execShouldFail(t, db, "DELETE audit.audit_log (append-only violation)",
		`DELETE FROM audit.audit_log WHERE resource = 'artifact' AND resource_id = 'art-1'`)

	t.Log("PHYSICAL PASS: Audit WORM triggers -?UPDATE and DELETE rejected by PostgreSQL trigger")
}

func TestPhysical007_EvidenceLedger_WOM_Triggers(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	evdID := "00000000-0000-0000-0000-000000000031"

	execShouldSucceed(t, db, "insert evidence for WORM test",
		`INSERT INTO evidence.evidence_ledger (evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ($1, 'chain-e', 1, '0000000000000000000000000000000000000000000000000000000000000000', 'eee0000000000000000000000000000000000000000000000000000000000000', 'mandatory', '{}', 'evt-e', 'tx-e', $2, 'user-e')`,
		evdID, tenantA)

	execShouldFail(t, db, "UPDATE evidence.evidence_ledger (append-only violation)",
		`UPDATE evidence.evidence_ledger SET evidence_type = 'decision' WHERE evidence_id = $1`, evdID)

	execShouldFail(t, db, "DELETE evidence.evidence_ledger (append-only violation)",
		`DELETE FROM evidence.evidence_ledger WHERE evidence_id = $1`, evdID)

	t.Log("PHYSICAL PASS: Evidence Ledger WORM triggers -?UPDATE and DELETE rejected by PostgreSQL trigger")
}

func TestPhysical007_RevokedTokens_RLS(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	execShouldSucceed(t, db, "insert revoked token A",
		`INSERT INTO security.revoked_tokens (token_jti, tenant_id, revoked_by, expires_at, reason)
		 VALUES ('jti-a', $1, 'admin-a', now() + interval '1 hour', 'user-logout')`,
		tenantA)

	execShouldSucceed(t, db, "insert revoked token B",
		`INSERT INTO security.revoked_tokens (token_jti, tenant_id, revoked_by, expires_at, reason)
		 VALUES ('jti-b', $1, 'admin-b', now() + interval '1 hour', 'user-logout')`,
		tenantB)

	countA := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM security.revoked_tokens`)
	if countA != 1 {
		t.Errorf("Tenant A should see 1 revoked token, got %d", countA)
	}

	countB := countRowsAsTenant(t, db, tenantB, `SELECT count(*) FROM security.revoked_tokens`)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 revoked token, got %d", countB)
	}

	t.Log("PHYSICAL PASS: Revoked tokens RLS -?tenant isolation verified")
}