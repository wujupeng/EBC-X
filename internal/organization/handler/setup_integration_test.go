package handler


import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/google/uuid"

	"github.com/wujupeng/ebcx/internal/organization"
	"github.com/wujupeng/ebcx/internal/organization/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/organization/repository"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

var (
	orgSuperDB  *sql.DB
	orgAppDB    *sql.DB
	orgSharedPG *embeddedpostgres.EmbeddedPostgres

	orgRlsManager     *security.RLSManager
	orgUow            repository.UnitOfWork
	orgRepo           *repository.OrganizationRepositoryPostgreSQL
	orgIdemRepo       *repository.CommandIdempotencyRepositoryPostgreSQL
	orgEvWriter       *evidence_adapter.OrganizationEvidenceWriterImpl
	orgStructEvWriter *evidence_adapter.OrganizationTreeStructuralEvidenceWriterImpl
	orgOutboxPub      *outbox.Publisher
	orgCreateHandler  *CreateOrganizationHandler
	orgUpdateHandler  *UpdateOrganizationHandler
	orgMoveHandler    *MoveOrganizationHandler
	orgCoordinator    *organization.OrganizationTreeCoordinator
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5440).
			Database("ebcx_organization_itest").
			Username("postgres").
			Password("itest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	superConnStr := "host=localhost port=5440 user=postgres password=itest dbname=ebcx_organization_itest sslmode=disable"
	sdb, err := sql.Open("postgres", superConnStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := findOrgMigrationDir()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := runOrgMigrations(sdb, migrationDir); err != nil {
		fmt.Printf("failed to run migrations: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	_, err = sdb.Exec(`CREATE ROLE ebcx_app LOGIN PASSWORD 'apppass' IN ROLE ebcx_runtime`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		fmt.Printf("failed to create ebcx_app role: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	appConnStr := "host=localhost port=5440 user=ebcx_app password=apppass dbname=ebcx_organization_itest sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	orgSuperDB = sdb
	orgAppDB = adb
	orgSharedPG = pg

	orgRlsManager = security.NewRLSManager(adb)
	orgUow = repository.NewUnitOfWorkPostgreSQL(orgRlsManager)
	orgRepo = repository.NewOrganizationRepositoryPostgreSQL()
	orgIdemRepo = repository.NewCommandIdempotencyRepositoryPostgreSQL()
	orgEvWriter = evidence_adapter.NewOrganizationEvidenceWriter("test-runner")
	orgStructEvWriter = evidence_adapter.NewOrganizationTreeStructuralEvidenceWriter("test-runner")
	orgOutboxPub = outbox.NewPublisher(adb)
	orgCreateHandler = NewCreateOrganizationHandler(orgUow, orgRepo, orgEvWriter, orgOutboxPub, orgIdemRepo)
	orgUpdateHandler = NewUpdateOrganizationHandler(orgUow, orgRepo, orgEvWriter, orgOutboxPub, orgIdemRepo)

	treeRepoAdapter := repository.NewTreeCoordinatorRepositoryAdapter(orgRepo)
	treeEvAdapter := evidence_adapter.NewTreeCoordinatorEvidenceWriterAdapter(orgEvWriter)
	treeStructEvAdapter := evidence_adapter.NewTreeCoordinatorStructuralEvidenceWriterAdapter(orgStructEvWriter)
	orgCoordinator = organization.NewOrganizationTreeCoordinator(treeRepoAdapter, treeEvAdapter, treeStructEvAdapter, orgOutboxPub)
	orgMoveHandler = NewMoveOrganizationHandler(orgUow, orgRepo, orgEvWriter, orgStructEvWriter, orgIdemRepo, orgCoordinator)

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func findOrgMigrationDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "db", "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("db/migrations not found")
}

func runOrgMigrations(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return orgMigrationVersion(files[i]) < orgMigrationVersion(files[j])
	})
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	return nil
}

func orgMigrationVersion(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func cleanupOrgTables(t *testing.T) {
	t.Helper()
	_, err := orgSuperDB.Exec(`TRUNCATE business.command_idempotency, business.organizations, business.enterprises, evidence.evidence_ledger, outbox.events CASCADE`)
	if err != nil {
		t.Fatalf("failed to cleanup tables: %v", err)
	}
}

func countOrgRows(t *testing.T, table string) int {
	t.Helper()
	var count int
	err := orgSuperDB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}
	return count
}

func createTestEnterprise(t *testing.T, tenantID string) string {
	t.Helper()
	entID := uuid.NewString()
	_, err := orgSuperDB.Exec(`
		INSERT INTO business.enterprises (enterprise_id, name, tenant_id)
		VALUES ($1, $2, $3)
	`, entID, "TestEnterprise-"+entID[:8], tenantID)
	if err != nil {
		t.Fatalf("failed to create test enterprise: %v", err)
	}
	return entID
}

var orgTestCtx = context.Background()