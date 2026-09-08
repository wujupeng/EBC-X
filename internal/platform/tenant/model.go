package tenant

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidCountryPack  = errors.New("invalid country pack")
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrPackNotEnabled      = errors.New("country pack not enabled for tenant")
	ErrCoreTableFork       = errors.New("core table structure forked across country packs")
	ErrMissingTenantHeader = errors.New("missing X-Tenant-Id header")
	ErrMissingPackHeader   = errors.New("missing X-Country-Pack header")
)

type CountryPack string

const (
	PackChina CountryPack = "CN"
	PackEU    CountryPack = "EU"
	PackUS    CountryPack = "US"
	PackJapan CountryPack = "JP"
	PackASEAN CountryPack = "ASEAN"
)

var AllCountryPacks = []CountryPack{PackChina, PackEU, PackUS, PackJapan, PackASEAN}

func IsValidCountryPack(pack CountryPack) bool {
	for _, p := range AllCountryPacks {
		if p == pack {
			return true
		}
	}
	return false
}

type Tenant struct {
	TenantID     string        `json:"tenantId"`
	TenantName   string        `json:"tenantName"`
	CountryCode  string        `json:"countryCode"`
	EnabledPacks []CountryPack `json:"enabledPacks"`
	CreatedAt    time.Time     `json:"createdAt"`
	IsActive     bool          `json:"isActive"`
}

type PackExtension struct {
	PackName       CountryPack `json:"packName"`
	DisplayName    string      `json:"displayName"`
	ExtTableSuffix string      `json:"extTableSuffix"`
	IsActive       bool        `json:"isActive"`
}

var PackExtensions = []PackExtension{
	{PackChina, "China Pack", "cn_ext", true},
	{PackEU, "European Union Pack", "eu_ext", true},
	{PackUS, "United States Pack", "us_ext", true},
	{PackJapan, "Japan Pack", "jp_ext", true},
	{PackASEAN, "ASEAN Pack", "asean_ext", true},
}

func GetPackExtension(pack CountryPack) (*PackExtension, error) {
	for _, ext := range PackExtensions {
		if ext.PackName == pack {
			return &ext, nil
		}
	}
	return nil, ErrInvalidCountryPack
}

func ExtensionTableName(baseTable string, pack CountryPack) (string, error) {
	ext, err := GetPackExtension(pack)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", baseTable, ext.ExtTableSuffix), nil
}

func (t *Tenant) IsPackEnabled(pack CountryPack) bool {
	for _, p := range t.EnabledPacks {
		if p == pack {
			return true
		}
	}
	return false
}

type CoreTableSpec struct {
	TableName string
	Columns   []ColumnSpec
}

type ColumnSpec struct {
	Name     string
	Type     string
	Nullable bool
	IsCore   bool
}

func ValidateCoreTableNoFork(specs []CoreTableSpec) error {
	if len(specs) < 2 {
		return nil
	}

	base := specs[0]
	for i := 1; i < len(specs); i++ {
		other := specs[i]
		if len(base.Columns) != len(other.Columns) {
			return fmt.Errorf("%w: %s has %d columns, %s has %d columns",
				ErrCoreTableFork, base.TableName, len(base.Columns), other.TableName, len(other.Columns))
		}
		for j, bc := range base.Columns {
			oc := other.Columns[j]
			if bc.Name != oc.Name || bc.Type != oc.Type {
				return fmt.Errorf("%w: column %d mismatch (%s.%s vs %s.%s)",
					ErrCoreTableFork, j, base.TableName, bc.Name, other.TableName, oc.Name)
			}
		}
	}
	return nil
}
