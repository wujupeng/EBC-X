package artifact

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ArtifactType string

const (
	ArtifactPDF           ArtifactType = "pdf"
	ArtifactImage         ArtifactType = "image"
	ArtifactCAD           ArtifactType = "cad"
	ArtifactQualityReport ArtifactType = "quality_report"
	ArtifactContract      ArtifactType = "contract"
	ArtifactInvoice       ArtifactType = "invoice"
	ArtifactEvidence      ArtifactType = "evidence"
	ArtifactLog           ArtifactType = "log"
	ArtifactCustom        ArtifactType = "custom"
)

var AllArtifactTypes = []ArtifactType{
	ArtifactPDF, ArtifactImage, ArtifactCAD, ArtifactQualityReport,
	ArtifactContract, ArtifactInvoice, ArtifactEvidence, ArtifactLog, ArtifactCustom,
}

func IsValidArtifactType(t ArtifactType) bool {
	for _, a := range AllArtifactTypes {
		if a == t {
			return true
		}
	}
	return false
}

type Artifact struct {
	ArtifactID   string       `json:"artifactId"`
	TenantID     string       `json:"tenantId"`
	EvidenceID   string       `json:"evidenceId"`
	ArtifactType ArtifactType `json:"artifactType"`
	Version      int64        `json:"version"`
	Filename     string       `json:"filename"`
	ContentType  string       `json:"contentType"`
	SizeBytes    int64        `json:"sizeBytes"`
	Checksum     string       `json:"checksum"`
	UploadedAt   time.Time    `json:"uploadedAt"`
	UploadedBy   string       `json:"uploadedBy"`
	ObjectKey    string       `json:"objectKey"`
}

type ArtifactMetadata struct {
	ArtifactID    string       `json:"artifactId"`
	TenantID      string       `json:"tenantId"`
	EvidenceID    string       `json:"evidenceId"`
	ArtifactType  ArtifactType `json:"artifactType"`
	Version       int64        `json:"version"`
	Filename      string       `json:"filename"`
	ContentType   string       `json:"contentType"`
	SizeBytes     int64        `json:"sizeBytes"`
	Checksum      string       `json:"checksum"`
	UploadedAt    time.Time    `json:"uploadedAt"`
	UploadedBy    string       `json:"uploadedBy"`
	ObjectKey     string       `json:"objectKey"`
	WORMProtected bool         `json:"wormProtected"`
}

var (
	ErrArtifactAlreadyExists   = errors.New("artifact already exists (WORM violation — write once)")
	ErrArtifactNotFound        = errors.New("artifact not found")
	ErrArtifactWORMDelete      = errors.New("artifact is WORM-protected — delete forbidden")
	ErrArtifactWORMModify      = errors.New("artifact is WORM-protected — modify forbidden")
	ErrInvalidNamingConvention = errors.New("invalid artifact naming convention")
	ErrInvalidArtifactType     = errors.New("invalid artifact type")
	ErrCrossTenantAccess       = errors.New("cross-tenant artifact access forbidden")
	ErrMissingEvidenceRef      = errors.New("artifact missing evidenceId — Evidence First violation")
)

func BuildObjectKey(tenantID, evidenceID string, artifactType ArtifactType, version int64, filename string) string {
	return fmt.Sprintf("%s/%s/%s/%d/%s", tenantID, evidenceID, string(artifactType), version, filename)
}

func ParseObjectKey(key string) (tenantID, evidenceID string, artifactType ArtifactType, version int64, filename string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 5 {
		return "", "", "", 0, "", ErrInvalidNamingConvention
	}
	tenantID = parts[0]
	evidenceID = parts[1]
	artifactType = ArtifactType(parts[2])
	version = 0
	fmt.Sscanf(parts[3], "%d", &version)
	filename = parts[4]

	if tenantID == "" || evidenceID == "" || filename == "" {
		return "", "", "", 0, "", ErrInvalidNamingConvention
	}
	if !IsValidArtifactType(artifactType) {
		return "", "", "", 0, "", ErrInvalidArtifactType
	}
	if version < 1 {
		return "", "", "", 0, "", ErrInvalidNamingConvention
	}
	return tenantID, evidenceID, artifactType, version, filename, nil
}

func (a *Artifact) Validate() error {
	if a.TenantID == "" {
		return errors.New("artifact missing tenantId")
	}
	if a.EvidenceID == "" {
		return ErrMissingEvidenceRef
	}
	if !IsValidArtifactType(a.ArtifactType) {
		return ErrInvalidArtifactType
	}
	if a.Version < 1 {
		return errors.New("artifact version must be >= 1")
	}
	if a.Filename == "" {
		return errors.New("artifact missing filename")
	}
	expectedKey := BuildObjectKey(a.TenantID, a.EvidenceID, a.ArtifactType, a.Version, a.Filename)
	if a.ObjectKey != "" && a.ObjectKey != expectedKey {
		return ErrInvalidNamingConvention
	}
	return nil
}

func (a *Artifact) ObjectKeyCanonical() string {
	return BuildObjectKey(a.TenantID, a.EvidenceID, a.ArtifactType, a.Version, a.Filename)
}
