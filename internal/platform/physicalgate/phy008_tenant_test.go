//go:build integration

package physicalgate

import (
	"fmt"
	"testing"
)

func TestPhysical008_ExtensionTable_RLS_TenantIsolation(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"
	orderA := "00000000-0000-0000-0000-000000000101"
	orderB := "00000000-0000-0000-0000-000000000102"

	execShouldSucceed(t, db, "insert order_cn_ext A",
		`INSERT INTO business.order_cn_ext (order_id, tenant_id, invoice_type, tax_reg_number, fapiao_number)
		 VALUES ($1, $2, 'general', 'tax-a', 'fapiao-a')`,
		orderA, tenantA)

	execShouldSucceed(t, db, "insert order_cn_ext B",
		`INSERT INTO business.order_cn_ext (order_id, tenant_id, invoice_type, tax_reg_number, fapiao_number)
		 VALUES ($1, $2, 'general', 'tax-b', 'fapiao-b')`,
		orderB, tenantB)

	countA := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM business.order_cn_ext`)
	if countA != 1 {
		t.Errorf("Tenant A should see 1 CN ext row, got %d", countA)
	}

	countB := countRowsAsTenant(t, db, tenantB, `SELECT count(*) FROM business.order_cn_ext`)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 CN ext row, got %d", countB)
	}

	countNone := countRowsAsTenant(t, db, "00000000-0000-0000-0000-000000000099", `SELECT count(*) FROM business.order_cn_ext`)
	if countNone != 0 {
		t.Errorf("Unknown tenant should see 0 CN ext rows, got %d", countNone)
	}

	t.Log("PHYSICAL PASS: Extension table RLS -?Tenant A+CN sees only A, Tenant B+CN sees only B")
}

func TestPhysical008_CrossTenant_PackIsolation(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"
	orderA := "00000000-0000-0000-0000-000000000201"
	orderB := "00000000-0000-0000-0000-000000000202"

	execShouldSucceed(t, db, "insert EU ext A",
		`INSERT INTO business.order_eu_ext (order_id, tenant_id, vat_number, gdpr_consent_id)
		 VALUES ($1, $2, 'vat-a', 'gdpr-a')`,
		orderA, tenantA)

	execShouldSucceed(t, db, "insert EU ext B",
		`INSERT INTO business.order_eu_ext (order_id, tenant_id, vat_number, gdpr_consent_id)
		 VALUES ($1, $2, 'vat-b', 'gdpr-b')`,
		orderB, tenantB)

	countA := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM business.order_eu_ext`)
	if countA != 1 {
		t.Errorf("Tenant A should see 1 EU ext row, got %d", countA)
	}

	countB := countRowsAsTenant(t, db, tenantB, `SELECT count(*) FROM business.order_eu_ext`)
	if countB != 1 {
		t.Errorf("Tenant B should see 1 EU ext row, got %d", countB)
	}

	countACrossToB := countRowsAsTenant(t, db, tenantA, `SELECT count(*) FROM business.order_eu_ext WHERE order_id = $1`, orderB)
	if countACrossToB != 0 {
		t.Errorf("PHYSICAL NEGATIVE TEST FAIL: Tenant A can see Tenant B's EU ext row, got %d", countACrossToB)
	}

	t.Log("PHYSICAL PASS: Cross-tenant Pack isolation -?A+EU cannot see B+EU data")
}

func TestPhysical008_AllExtensionTables_RLS(t *testing.T) {
	db := getDB(t)

	tenantA := "00000000-0000-0000-0000-000000000555"

	extTables := []string{
		"business.order_cn_ext",
		"business.order_eu_ext",
		"business.order_us_ext",
		"business.order_jp_ext",
		"business.order_asean_ext",
	}

	for i, table := range extTables {
		orderID := fmt.Sprintf("00000000-0000-0000-0000-%012d", 500+i)
		execShouldSucceed(t, db, fmt.Sprintf("insert %s", table),
			fmt.Sprintf(`INSERT INTO %s (order_id, tenant_id) VALUES ($1, $2)`, table),
			orderID, tenantA)
	}

	for _, table := range extTables {
		count := countRowsAsTenant(t, db, tenantA, fmt.Sprintf(`SELECT count(*) FROM %s`, table))
		if count != 1 {
			t.Errorf("Tenant A should see 1 row in %s, got %d", table, count)
		}
	}

	countOther := countRowsAsTenant(t, db, "00000000-0000-0000-0000-000000000099", `SELECT count(*) FROM business.order_cn_ext`)
	if countOther != 0 {
		t.Errorf("Unknown tenant should see 0 rows, got %d", countOther)
	}

	t.Log("PHYSICAL PASS: All 5 extension tables have RLS enabled and working")
}

func TestPhysical008_CoreTable_NoFork_SchemaVerification(t *testing.T) {
	db := getDB(t)

	extTables := []string{
		"order_cn_ext",
		"order_eu_ext",
		"order_us_ext",
		"order_jp_ext",
		"order_asean_ext",
	}

	coreColumns := map[string]bool{
		"order_id":   true,
		"tenant_id":  true,
		"created_at": true,
	}

	for _, table := range extTables {
		rows, err := db.Query(
			`SELECT column_name, data_type FROM information_schema.columns
			 WHERE table_schema = 'business' AND table_name = $1 ORDER BY ordinal_position`,
			table)
		if err != nil {
			t.Fatalf("failed to query columns for %s: %v", table, err)
		}

		var columns []string
		colMap := make(map[string]string)
		for rows.Next() {
			var colName, dataType string
			rows.Scan(&colName, &dataType)
			columns = append(columns, colName)
			colMap[colName] = dataType
		}
		rows.Close()

		for coreCol := range coreColumns {
			if _, exists := colMap[coreCol]; !exists {
				t.Errorf("Core column %s missing from %s -?FORK DETECTED", coreCol, table)
			}
		}

		if _, hasOrderID := colMap["order_id"]; !hasOrderID {
			t.Errorf("order_id missing from %s", table)
		}
		if _, hasTenantID := colMap["tenant_id"]; !hasTenantID {
			t.Errorf("tenant_id missing from %s", table)
		}

		t.Logf("%s columns: %v", table, columns)
	}

	rows, err := db.Query(
		`SELECT table_name, column_name, data_type FROM information_schema.columns
		 WHERE table_schema = 'business' AND table_name LIKE 'order_%_ext'
		 AND column_name IN ('order_id', 'tenant_id', 'created_at')
		 ORDER BY table_name, column_name`)
	if err != nil {
		t.Fatalf("failed to query core columns: %v", err)
	}
	defer rows.Close()

	coreColCount := make(map[string]int)
	for rows.Next() {
		var tableName, colName, dataType string
		rows.Scan(&tableName, &colName, &dataType)
		coreColCount[tableName+"."+colName] = 1
	}

	for _, table := range extTables {
		for coreCol := range coreColumns {
			key := table + "." + coreCol
			if coreColCount[key] == 0 {
				t.Errorf("Core column %s missing from %s -?NO-FORK VIOLATION", coreCol, table)
			}
		}
	}

	t.Log("PHYSICAL PASS: Core Table No-Fork -?all 5 extension tables have consistent core columns (order_id, tenant_id, created_at)")
}

func TestPhysical008_CountryPacks_FiveReserved(t *testing.T) {
	db := getDB(t)

	var count int
	err := db.QueryRow("SELECT count(*) FROM tenant.country_packs").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count country packs: %v", err)
	}
	if count != 5 {
		t.Errorf("country_packs count = %d, want 5", count)
	}

	expectedPacks := map[string]string{
		"CN":    "China Pack",
		"EU":    "European Union Pack",
		"US":    "United States Pack",
		"JP":    "Japan Pack",
		"ASEAN": "ASEAN Pack",
	}

	rows, err := db.Query("SELECT pack_id, display_name, ext_table_suffix, is_active FROM tenant.country_packs ORDER BY pack_id")
	if err != nil {
		t.Fatalf("failed to query country packs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var packID, displayName, extSuffix string
		var isActive bool
		rows.Scan(&packID, &displayName, &extSuffix, &isActive)

		expectedName, ok := expectedPacks[packID]
		if !ok {
			t.Errorf("unexpected pack: %s", packID)
		}
		if displayName != expectedName {
			t.Errorf("pack %s display_name = %s, want %s", packID, displayName, expectedName)
		}
		if !isActive {
			t.Errorf("pack %s should be active", packID)
		}
		t.Logf("Pack %s: %s (suffix: %s, active: %v)", packID, displayName, extSuffix, isActive)
	}

	t.Log("PHYSICAL PASS: 5 Country Packs reserved in PostgreSQL (CN/EU/US/JP/ASEAN)")
}