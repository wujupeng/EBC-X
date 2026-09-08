//go:build integration

package physicalgate

import (

	"testing"
)

func TestPhysical006_Artifact_RLS_TenantIsolation(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"
	evdA := "00000000-0000-0000-0000-000000000011"
	evdB := "00000000-0000-0000-0000-000000000012"

	execShouldSucceed(t, db, "insert evidence A",
		`INSERT INTO evidence.evidence_ledger (evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ($1, 'chain-1', 1, '0000000000000000000000000000000000000000000000000000000000000000', 'aaa0000000000000000000000000000000000000000000000000000000000000', 'mandatory', '{}', 'evt-1', 'tx-1', $2, 'user-a')`,
		evdA, tenantA)

	execShouldSucceed(t, db, "insert evidence B",
		`INSERT INTO evidence.evidence_ledger (evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ($1, 'chain-2', 1, '0000000000000000000000000000000000000000000000000000000000000000', 'bbb0000000000000000000000000000000000000000000000000000000000000', 'mandatory', '{}', 'evt-2', 'tx-2', $2, 'user-b')`,
		evdB, tenantB)

	execShouldSucceed(t, db, "insert artifact A",
		`INSERT INTO artifact.metadata (tenant_id, evidence_id, artifact_type, version, filename, content_type, size_bytes, checksum, object_key, uploaded_by)
		 VALUES ($1, $2, 'pdf', 1, 'report-a.pdf', 'application/pdf', 100, 'checksum-a', 'tenant-a/evd-a/pdf/1/report-a.pdf', 'user-a')`,
		tenantA, evdA)

	execShouldSucceed(t, db, "insert artifact B",
		`INSERT INTO artifact.metadata (tenant_id, evidence_id, artifact_type, version, filename, content_type, size_bytes, checksum, object_key, uploaded_by)
		 VALUES ($1, $2, 'pdf', 1, 'report-b.pdf', 'application/pdf', 200, 'checksum-b', 'tenant-b/evd-b/pdf/1/report-b.pdf', 'user-b')`,
		tenantB, evdB)

	countA := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM artifact.metadata`)
	if countA != 1 {
		t.Errorf("Tenant A should see 1 artifact, got %d", countA)
	}

	countB := countRowsAsTenant(t, db, tenantB, `SELECT count(*) FROM artifact.metadata`)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 artifact, got %d", countB)
	}

	countNone := countRowsAsTenant(t, db, "00000000-0000-0000-0000-000000000099", `SELECT count(*) FROM artifact.metadata`)
	if countNone != 0 {
		t.Errorf("Unknown tenant should see 0 artifacts, got %d", countNone)
	}

	t.Log("PHYSICAL PASS: Artifact RLS - Tenant A sees only A, Tenant B sees only B, unknown tenant sees 0")
}

func TestPhysical006_Artifact_WORM_Triggers(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	evdA := "00000000-0000-0000-0000-000000000041"

	execShouldSucceed(t, db, "insert evidence for WORM test",
		`INSERT INTO evidence.evidence_ledger (evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ($1, 'chain-w', 1, '0000000000000000000000000000000000000000000000000000000000000000', 'www0000000000000000000000000000000000000000000000000000000000000', 'mandatory', '{}', 'evt-w', 'tx-w', $2, 'user-w')`,
		evdA, tenantA)

	execShouldSucceed(t, db, "insert artifact for WORM test",
		`INSERT INTO artifact.metadata (tenant_id, evidence_id, artifact_type, version, filename, content_type, size_bytes, checksum, object_key, uploaded_by)
		 VALUES ($1, $2, 'pdf', 1, 'worm-test.pdf', 'application/pdf', 50, 'worm-checksum', 'tenant-w/evd-w/pdf/1/worm-test.pdf', 'user-w')`,
		tenantA, evdA)

	execShouldFail(t, db, "UPDATE artifact.metadata (WORM violation)",
		`UPDATE artifact.metadata SET filename = 'modified.pdf' WHERE checksum = 'worm-checksum'`)

	execShouldFail(t, db, "DELETE artifact.metadata (WORM violation)",
		`DELETE FROM artifact.metadata WHERE checksum = 'worm-checksum'`)

	t.Log("PHYSICAL PASS: Artifact WORM triggers - UPDATE and DELETE rejected by PostgreSQL trigger")
}

func TestPhysical006_Artifact_FK_EvidenceRef(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	validEvd := "00000000-0000-0000-0000-000000000021"
	invalidEvd := "00000000-0000-0000-0000-000000000099"

	execShouldSucceed(t, db, "insert evidence for FK test",
		`INSERT INTO evidence.evidence_ledger (evidence_id, chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ($1, 'chain-fk', 1, '0000000000000000000000000000000000000000000000000000000000000000', 'fkk0000000000000000000000000000000000000000000000000000000000000', 'mandatory', '{}', 'evt-fk', 'tx-fk', $2, 'user-fk')`,
		validEvd, tenantA)

	execShouldSucceed(t, db, "insert artifact with valid evidence_id",
		`INSERT INTO artifact.metadata (tenant_id, evidence_id, artifact_type, version, filename, content_type, size_bytes, checksum, object_key, uploaded_by)
		 VALUES ($1, $2, 'pdf', 1, 'fk-valid.pdf', 'application/pdf', 10, 'fk-valid-checksum', 'tenant-fk/evd-fk/pdf/1/fk-valid.pdf', 'user-fk')`,
		tenantA, validEvd)

	execShouldFail(t, db, "insert artifact with invalid evidence_id (FK violation)",
		`INSERT INTO artifact.metadata (tenant_id, evidence_id, artifact_type, version, filename, content_type, size_bytes, checksum, object_key, uploaded_by)
		 VALUES ($1, $2, 'pdf', 1, 'fk-invalid.pdf', 'application/pdf', 10, 'fk-invalid-checksum', 'tenant-fk/evd-invalid/pdf/1/fk-invalid.pdf', 'user-fk')`,
		tenantA, invalidEvd)

	t.Log("PHYSICAL PASS: Artifact FK - valid evidence_id accepted, invalid evidence_id rejected by PostgreSQL FK constraint")
}
