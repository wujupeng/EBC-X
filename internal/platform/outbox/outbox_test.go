package outbox

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/wujupeng/ebcx/internal/evidence"
)

func setupOutboxDB(t *testing.T) (*embeddedpostgres.EmbeddedPostgres, *sql.DB) {
	t.Helper()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().Username("postgres").Password("postgres").Database("ebcx_outbox_test").Port(5434),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	connStr := "host=localhost port=5434 user=postgres password=postgres dbname=ebcx_outbox_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	repoRoot, _ := os.Getwd()
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}
	for _, f := range []string{"V1__create_schemas.sql", "V2__create_roles.sql", "V3__tenant_foundation.sql", "V4__evidence_ledger.sql", "V5__outbox.sql"} {
		content, err := os.ReadFile(filepath.Join(repoRoot, "db", "migrations", f))
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("exec %s: %v", f, err)
		}
	}
	db.Exec("CREATE ROLE ebcx_app LOGIN PASSWORD 'apppass' IN ROLE ebcx_runtime")
	return pg, db
}

func TestFaultIsolation_EventBusDown_TransactionContinues(t *testing.T) {
	pg, db := setupOutboxDB(t)
	defer pg.Stop()
	defer db.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	genHash := evidence.GenesisHash("fault-test")
	firstHash := evidence.ChainHash("fault-test", 1, genHash, []byte(`{}`), "evt-1", "tx-1")
	_, err = tx.ExecContext(ctx,
		`INSERT INTO evidence.evidence_ledger (chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
		 VALUES ('fault-test', 1, $2, $3, 'mandatory', '{}', 'evt-1', 'tx-1', $1::uuid, 'test')`, tenantID, genHash, firstHash)
	if err != nil {
		t.Fatalf("evidence insert: %v", err)
	}

	pub := NewPublisher(db)
	payload := []byte(`{"order_id":"ORD-001"}`)
	if err := pub.Write(ctx, tx, "Order", "ORD-001", "OrderCreated", tenantID, "corr-1", "", payload); err != nil {
		t.Fatalf("outbox write: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	t.Log("✅ Transaction committed with Evidence + Outbox in same transaction")

	bus := NewMemoryBus()
	bus.SetAvailable(false)

	n, err := pub.PublishPending(ctx, bus)
	if err == nil {
		t.Fatal("expected ErrEventBusUnavailable")
	}
	t.Logf("✅ EventBus unavailable — %d published, outbox accumulating", n)

	var pendingCount int
	db.QueryRow("SELECT count(*) FROM outbox.events WHERE status='pending'").Scan(&pendingCount)
	if pendingCount != 1 {
		t.Fatalf("pending = %d, want 1", pendingCount)
	}
	t.Log("✅ Outbox accumulating — event retained for replay")

	var evCount int
	db.QueryRow("SELECT count(*) FROM evidence.evidence_ledger").Scan(&evCount)
	if evCount != 1 {
		t.Fatalf("evidence = %d, want 1", evCount)
	}
	t.Log("✅ Evidence Ledger intact — Transaction Core not affected by EventBus outage")

	if pub.Mode() != ModeDegraded {
		t.Fatalf("mode = %s, want degraded", pub.Mode())
	}
	t.Log("✅ Mode → DEGRADED")

	bus.SetAvailable(true)
	n, err = pub.PublishPending(ctx, bus)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("replay = %d, want 1", n)
	}
	t.Log("✅ EventBus recovered — Outbox replayed pending event")

	var pubCount int
	db.QueryRow("SELECT count(*) FROM outbox.events WHERE status='published'").Scan(&pubCount)
	if pubCount != 1 {
		t.Fatalf("published = %d, want 1", pubCount)
	}
	t.Log("✅ Event marked published after replay")
}

func TestIdempotentConsumer(t *testing.T) {
	consumer := NewIdempotentConsumer()
	e := Event{EventID: "evt-001"}

	if err := consumer.Consume(e); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if err := consumer.Consume(e); err != ErrAlreadyConsumed {
		t.Fatal("duplicate consume not rejected")
	}
	if consumer.ConsumedCount() != 1 {
		t.Fatalf("consumed = %d, want 1", consumer.ConsumedCount())
	}
	t.Log("✅ Idempotent consumer — duplicate event rejected")
}

func TestSameTransactionCommit(t *testing.T) {
	pg, db := setupOutboxDB(t)
	defer pg.Stop()
	defer db.Close()

	ctx := context.Background()
	tenantID := "00000000-0000-0000-0000-000000000001"

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	pub := NewPublisher(db)
	pub.Write(ctx, tx, "Order", "ORD-002", "OrderCreated", tenantID, "corr-2", "", []byte(`{"v":2}`))

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var count int
	db.QueryRow("SELECT count(*) FROM outbox.events WHERE aggregate_id='ORD-002'").Scan(&count)
	if count != 1 {
		t.Fatalf("outbox count = %d, want 1", count)
	}
	t.Log("✅ Outbox written in same transaction as business state")

	tx2, _ := db.BeginTx(ctx, nil)
	pub.Write(ctx, tx2, "Order", "ORD-003", "OrderCreated", tenantID, "corr-3", "", []byte(`{"v":3}`))
	tx2.Rollback()

	db.QueryRow("SELECT count(*) FROM outbox.events WHERE aggregate_id='ORD-003'").Scan(&count)
	if count != 0 {
		t.Fatalf("rolled-back outbox count = %d, want 0", count)
	}
	t.Log("✅ Transaction rollback discards outbox event (atomicity)")
}
