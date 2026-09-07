package tenant

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type PackRouter struct {
	tenants map[string]*Tenant
}

func NewPackRouter() *PackRouter {
	return &PackRouter{
		tenants: make(map[string]*Tenant),
	}
}

func (r *PackRouter) RegisterTenant(tenant *Tenant) {
	r.tenants[tenant.TenantID] = tenant
}

func (r *PackRouter) GetTenant(tenantID string) (*Tenant, error) {
	t, exists := r.tenants[tenantID]
	if !exists {
		return nil, ErrTenantNotFound
	}
	return t, nil
}

type PackRoute struct {
	TenantID       string
	CountryPack    CountryPack
	ExtensionTable string
	Tenant         *Tenant
}

func (r *PackRouter) Route(ctx context.Context, tenantID string, pack CountryPack) (*PackRoute, error) {
	tenant, err := r.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}

	if !IsValidCountryPack(pack) {
		return nil, ErrInvalidCountryPack
	}

	if !tenant.IsPackEnabled(pack) {
		return nil, ErrPackNotEnabled
	}

	extTable, err := ExtensionTableName("business", pack)
	if err != nil {
		return nil, err
	}

	return &PackRoute{
		TenantID:       tenantID,
		CountryPack:    pack,
		ExtensionTable: extTable,
		Tenant:         tenant,
	}, nil
}

func (r *PackRouter) RouteFromHeaders(ctx context.Context, headers http.Header) (*PackRoute, error) {
	tenantID := headers.Get("X-Tenant-Id")
	if tenantID == "" {
		return nil, ErrMissingTenantHeader
	}

	packStr := headers.Get("X-Country-Pack")
	if packStr == "" {
		return nil, ErrMissingPackHeader
	}

	pack := CountryPack(strings.ToUpper(packStr))
	return r.Route(ctx, tenantID, pack)
}

func (r *PackRouter) ListEnabledPacks(tenantID string) ([]CountryPack, error) {
	tenant, err := r.GetTenant(tenantID)
	if err != nil {
		return nil, err
	}
	return tenant.EnabledPacks, nil
}

func (r *PackRouter) ValidateAllPacksReserved() error {
	if len(PackExtensions) != 5 {
		return errors.New("expected exactly 5 Country Pack extension points")
	}
	expected := map[CountryPack]bool{PackChina: true, PackEU: true, PackUS: true, PackJapan: true, PackASEAN: true}
	for _, ext := range PackExtensions {
		if !expected[ext.PackName] {
			return errors.New("unexpected pack in extensions")
		}
		if !ext.IsActive {
			return errors.New("pack extension not active")
		}
	}
	return nil
}