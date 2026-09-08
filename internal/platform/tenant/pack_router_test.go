package tenant

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func testTenant() *Tenant {
	return &Tenant{
		TenantID:     "tenant-001",
		TenantName:   "Test Corp",
		CountryCode:  "CN",
		EnabledPacks: []CountryPack{PackChina, PackEU},
		CreatedAt:    time.Now(),
		IsActive:     true,
	}
}

func TestIsValidCountryPack(t *testing.T) {
	for _, p := range AllCountryPacks {
		if !IsValidCountryPack(p) {
			t.Errorf("IsValidCountryPack(%q) = false, want true", p)
		}
	}
	if IsValidCountryPack("XX") {
		t.Error("IsValidCountryPack(XX) should be false")
	}
}

func TestAllCountryPacks_Count(t *testing.T) {
	if len(AllCountryPacks) != 5 {
		t.Errorf("AllCountryPacks len = %d, want 5", len(AllCountryPacks))
	}
}

func TestGetPackExtension(t *testing.T) {
	ext, err := GetPackExtension(PackChina)
	if err != nil {
		t.Fatalf("GetPackExtension error: %v", err)
	}
	if ext.DisplayName != "China Pack" {
		t.Errorf("DisplayName = %s, want China Pack", ext.DisplayName)
	}
	if ext.ExtTableSuffix != "cn_ext" {
		t.Errorf("ExtTableSuffix = %s, want cn_ext", ext.ExtTableSuffix)
	}

	_, err = GetPackExtension("XX")
	if err != ErrInvalidCountryPack {
		t.Errorf("expected ErrInvalidCountryPack, got %v", err)
	}
}

func TestExtensionTableName(t *testing.T) {
	name, err := ExtensionTableName("order", PackChina)
	if err != nil {
		t.Fatalf("ExtensionTableName error: %v", err)
	}
	if name != "order_cn_ext" {
		t.Errorf("ExtensionTableName = %s, want order_cn_ext", name)
	}

	name, _ = ExtensionTableName("invoice", PackUS)
	if name != "invoice_us_ext" {
		t.Errorf("ExtensionTableName = %s, want invoice_us_ext", name)
	}
}

func TestTenant_IsPackEnabled(t *testing.T) {
	tenant := testTenant()
	if !tenant.IsPackEnabled(PackChina) {
		t.Error("PackChina should be enabled")
	}
	if tenant.IsPackEnabled(PackUS) {
		t.Error("PackUS should not be enabled")
	}
}

func TestPackRouter_Route_Success(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())

	ctx := context.Background()
	route, err := router.Route(ctx, "tenant-001", PackChina)
	if err != nil {
		t.Fatalf("Route error: %v", err)
	}
	if route.TenantID != "tenant-001" {
		t.Errorf("TenantID = %s", route.TenantID)
	}
	if route.CountryPack != PackChina {
		t.Errorf("CountryPack = %s", route.CountryPack)
	}
	if route.ExtensionTable != "business_cn_ext" {
		t.Errorf("ExtensionTable = %s, want business_cn_ext", route.ExtensionTable)
	}
}

func TestPackRouter_Route_TenantNotFound(t *testing.T) {
	router := NewPackRouter()
	ctx := context.Background()
	_, err := router.Route(ctx, "nonexistent", PackChina)
	if err != ErrTenantNotFound {
		t.Errorf("expected ErrTenantNotFound, got %v", err)
	}
}

func TestPackRouter_Route_InvalidPack(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())
	ctx := context.Background()
	_, err := router.Route(ctx, "tenant-001", "XX")
	if err != ErrInvalidCountryPack {
		t.Errorf("expected ErrInvalidCountryPack, got %v", err)
	}
}

func TestPackRouter_Route_PackNotEnabled(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())
	ctx := context.Background()
	_, err := router.Route(ctx, "tenant-001", PackUS)
	if err != ErrPackNotEnabled {
		t.Errorf("expected ErrPackNotEnabled, got %v", err)
	}
}

func TestPackRouter_RouteFromHeaders_Success(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())

	headers := http.Header{}
	headers.Set("X-Tenant-Id", "tenant-001")
	headers.Set("X-Country-Pack", "CN")

	ctx := context.Background()
	route, err := router.RouteFromHeaders(ctx, headers)
	if err != nil {
		t.Fatalf("RouteFromHeaders error: %v", err)
	}
	if route.CountryPack != PackChina {
		t.Errorf("CountryPack = %s, want CN", route.CountryPack)
	}
}

func TestPackRouter_RouteFromHeaders_MissingTenant(t *testing.T) {
	router := NewPackRouter()
	headers := http.Header{}
	headers.Set("X-Country-Pack", "CN")

	ctx := context.Background()
	_, err := router.RouteFromHeaders(ctx, headers)
	if err != ErrMissingTenantHeader {
		t.Errorf("expected ErrMissingTenantHeader, got %v", err)
	}
}

func TestPackRouter_RouteFromHeaders_MissingPack(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())

	headers := http.Header{}
	headers.Set("X-Tenant-Id", "tenant-001")

	ctx := context.Background()
	_, err := router.RouteFromHeaders(ctx, headers)
	if err != ErrMissingPackHeader {
		t.Errorf("expected ErrMissingPackHeader, got %v", err)
	}
}

func TestPackRouter_RouteFromHeaders_CaseInsensitive(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())

	headers := http.Header{}
	headers.Set("X-Tenant-Id", "tenant-001")
	headers.Set("X-Country-Pack", "cn")

	ctx := context.Background()
	route, err := router.RouteFromHeaders(ctx, headers)
	if err != nil {
		t.Fatalf("RouteFromHeaders error: %v", err)
	}
	if route.CountryPack != PackChina {
		t.Errorf("CountryPack = %s, want CN (case-insensitive)", route.CountryPack)
	}
}

func TestPackRouter_ListEnabledPacks(t *testing.T) {
	router := NewPackRouter()
	router.RegisterTenant(testTenant())

	packs, err := router.ListEnabledPacks("tenant-001")
	if err != nil {
		t.Fatalf("ListEnabledPacks error: %v", err)
	}
	if len(packs) != 2 {
		t.Errorf("packs len = %d, want 2", len(packs))
	}
}

func TestValidateAllPacksReserved(t *testing.T) {
	router := NewPackRouter()
	err := router.ValidateAllPacksReserved()
	if err != nil {
		t.Errorf("ValidateAllPacksReserved error: %v", err)
	}
}

func TestValidateCoreTableNoFork_Identical(t *testing.T) {
	specs := []CoreTableSpec{
		{TableName: "order", Columns: []ColumnSpec{
			{Name: "id", Type: "uuid", IsCore: true},
			{Name: "tenant_id", Type: "uuid", IsCore: true},
			{Name: "amount", Type: "numeric", IsCore: true},
		}},
		{TableName: "order", Columns: []ColumnSpec{
			{Name: "id", Type: "uuid", IsCore: true},
			{Name: "tenant_id", Type: "uuid", IsCore: true},
			{Name: "amount", Type: "numeric", IsCore: true},
		}},
	}
	if err := ValidateCoreTableNoFork(specs); err != nil {
		t.Errorf("identical specs should pass: %v", err)
	}
}

func TestValidateCoreTableNoFork_DifferentColumns(t *testing.T) {
	specs := []CoreTableSpec{
		{TableName: "order_cn", Columns: []ColumnSpec{
			{Name: "id", Type: "uuid", IsCore: true},
			{Name: "tenant_id", Type: "uuid", IsCore: true},
		}},
		{TableName: "order_eu", Columns: []ColumnSpec{
			{Name: "id", Type: "uuid", IsCore: true},
			{Name: "tenant_id", Type: "uuid", IsCore: true},
			{Name: "vat_number", Type: "text", IsCore: true},
		}},
	}
	err := ValidateCoreTableNoFork(specs)
	if err == nil {
		t.Error("forked specs should fail")
	}
}

func TestValidateCoreTableNoFork_DifferentTypes(t *testing.T) {
	specs := []CoreTableSpec{
		{TableName: "order_cn", Columns: []ColumnSpec{
			{Name: "amount", Type: "numeric", IsCore: true},
		}},
		{TableName: "order_us", Columns: []ColumnSpec{
			{Name: "amount", Type: "decimal", IsCore: true},
		}},
	}
	err := ValidateCoreTableNoFork(specs)
	if err == nil {
		t.Error("different types should fail")
	}
}

func TestPackExtensions_AllFiveReserved(t *testing.T) {
	if len(PackExtensions) != 5 {
		t.Fatalf("PackExtensions len = %d, want 5", len(PackExtensions))
	}
	expected := map[CountryPack]string{
		PackChina: "cn_ext",
		PackEU:    "eu_ext",
		PackUS:    "us_ext",
		PackJapan: "jp_ext",
		PackASEAN: "asean_ext",
	}
	for _, ext := range PackExtensions {
		suffix, ok := expected[ext.PackName]
		if !ok {
			t.Errorf("unexpected pack: %s", ext.PackName)
		}
		if ext.ExtTableSuffix != suffix {
			t.Errorf("Pack %s suffix = %s, want %s", ext.PackName, ext.ExtTableSuffix, suffix)
		}
		if !ext.IsActive {
			t.Errorf("Pack %s should be active", ext.PackName)
		}
	}
}
