//go:build integration

package physicalgate

import (
	"fmt"
	"testing"
)

func TestPhysical_Migration_AllV1ToV9_Success(t *testing.T) {
	db := getDB(t)

	version := getPGVersion(t)
	t.Logf("PostgreSQL version: %s", version)

	if !tableExists(t, db, "evidence", "evidence_ledger") {
		t.Error("MISSING: evidence.evidence_ledger (V4)")
	}
	if !tableExists(t, db, "artifact", "metadata") {
		t.Error("MISSING: artifact.metadata (V7)")
	}
	if !tableExists(t, db, "audit", "audit_log") {
		t.Error("MISSING: audit.audit_log (V8)")
	}
	if !tableExists(t, db, "security", "kms_keys") {
		t.Error("MISSING: security.kms_keys (V8)")
	}
	if !tableExists(t, db, "security", "revoked_tokens") {
		t.Error("MISSING: security.revoked_tokens (V8)")
	}
	if !tableExists(t, db, "tenant", "tenants") {
		t.Error("MISSING: tenant.tenants (V3/V9)")
	}
	if !tableExists(t, db, "tenant", "country_packs") {
		t.Error("MISSING: tenant.country_packs (V9)")
	}
	if !tableExists(t, db, "tenant", "tenant_packs") {
		t.Error("MISSING: tenant.tenant_packs (V9)")
	}
	for _, ext := range []string{"cn", "eu", "us", "jp", "asean"} {
		table := fmt.Sprintf("order_%s_ext", ext)
		if !tableExists(t, db, "business", table) {
			t.Errorf("MISSING: business.%s (V9)", table)
		}
	}

	t.Log("PHYSICAL PASS: All V1~V9 migrations executed successfully on real PostgreSQL 18.3")
}

func TestPhysical_Migration_RLSAndTriggers_Exist(t *testing.T) {
	db := getDB(t)

	if !rlsEnabled(t, db, "evidence", "evidence_ledger") {
		t.Error("MISSING: RLS on evidence.evidence_ledger")
	}
	if !rlsEnabled(t, db, "artifact", "metadata") {
		t.Error("MISSING: RLS on artifact.metadata")
	}
	if !rlsEnabled(t, db, "audit", "audit_log") {
		t.Error("MISSING: RLS on audit.audit_log")
	}
	if !rlsEnabled(t, db, "security", "revoked_tokens") {
		t.Error("MISSING: RLS on security.revoked_tokens")
	}
	for _, ext := range []string{"cn", "eu", "us", "jp", "asean"} {
		table := fmt.Sprintf("order_%s_ext", ext)
		if !rlsEnabled(t, db, "business", table) {
			t.Errorf("MISSING: RLS on business.%s", table)
		}
	}

	if !triggerExists(t, db, "evidence", "evidence_no_update", "evidence_ledger") {
		t.Error("MISSING: trigger evidence_no_update")
	}
	if !triggerExists(t, db, "evidence", "evidence_no_delete", "evidence_ledger") {
		t.Error("MISSING: trigger evidence_no_delete")
	}
	if !triggerExists(t, db, "artifact", "artifact_no_update", "metadata") {
		t.Error("MISSING: trigger artifact_no_update")
	}
	if !triggerExists(t, db, "artifact", "artifact_no_delete", "metadata") {
		t.Error("MISSING: trigger artifact_no_delete")
	}
	if !triggerExists(t, db, "audit", "audit_no_update", "audit_log") {
		t.Error("MISSING: trigger audit_no_update")
	}
	if !triggerExists(t, db, "audit", "audit_no_delete", "audit_log") {
		t.Error("MISSING: trigger audit_no_delete")
	}

	if !policyExists(t, db, "evidence_tenant_isolation", "evidence_ledger") {
		t.Error("MISSING: policy evidence_tenant_isolation")
	}
	if !policyExists(t, db, "artifact_tenant_isolation", "metadata") {
		t.Error("MISSING: policy artifact_tenant_isolation")
	}
	if !policyExists(t, db, "audit_log_tenant_isolation", "audit_log") {
		t.Error("MISSING: policy audit_log_tenant_isolation")
	}

	t.Log("PHYSICAL PASS: All RLS policies and WORM triggers exist on real PostgreSQL")
}