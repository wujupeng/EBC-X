package artifact

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testArtifact() Artifact {
	return Artifact{
		TenantID:     "tenant-001",
		EvidenceID:   "evd-001",
		ArtifactType: ArtifactPDF,
		Version:      1,
		Filename:     "report.pdf",
		ContentType:  "application/pdf",
		UploadedBy:   "user-001",
	}
}

func TestBuildObjectKey(t *testing.T) {
	key := BuildObjectKey("tenant-001", "evd-001", ArtifactPDF, 1, "report.pdf")
	expected := "tenant-001/evd-001/pdf/1/report.pdf"
	if key != expected {
		t.Errorf("BuildObjectKey = %q, want %q", key, expected)
	}
}

func TestParseObjectKey_Valid(t *testing.T) {
	key := "tenant-001/evd-001/pdf/1/report.pdf"
	tenantID, evidenceID, artType, version, filename, err := ParseObjectKey(key)
	if err != nil {
		t.Fatalf("ParseObjectKey error: %v", err)
	}
	if tenantID != "tenant-001" || evidenceID != "evd-001" || artType != ArtifactPDF || version != 1 || filename != "report.pdf" {
		t.Errorf("ParseObjectKey fields mismatch: %s/%s/%s/%d/%s", tenantID, evidenceID, artType, version, filename)
	}
}

func TestParseObjectKey_InvalidParts(t *testing.T) {
	_, _, _, _, _, err := ParseObjectKey("tenant/evd/pdf/report.pdf")
	if err != ErrInvalidNamingConvention {
		t.Errorf("expected ErrInvalidNamingConvention, got %v", err)
	}
}

func TestParseObjectKey_InvalidType(t *testing.T) {
	_, _, _, _, _, err := ParseObjectKey("tenant/evd/badtype/1/file.pdf")
	if err != ErrInvalidArtifactType {
		t.Errorf("expected ErrInvalidArtifactType, got %v", err)
	}
}

func TestParseObjectKey_InvalidVersion(t *testing.T) {
	_, _, _, _, _, err := ParseObjectKey("tenant/evd/pdf/0/file.pdf")
	if err != ErrInvalidNamingConvention {
		t.Errorf("expected ErrInvalidNamingConvention for version=0, got %v", err)
	}
}

func TestIsValidArtifactType(t *testing.T) {
	valid := []ArtifactType{ArtifactPDF, ArtifactImage, ArtifactCAD, ArtifactQualityReport, ArtifactContract, ArtifactInvoice, ArtifactEvidence, ArtifactLog, ArtifactCustom}
	for _, vt := range valid {
		if !IsValidArtifactType(vt) {
			t.Errorf("IsValidArtifactType(%q) = false, want true", vt)
		}
	}
	if IsValidArtifactType("badtype") {
		t.Error("IsValidArtifactType(badtype) = true, want false")
	}
}

func TestArtifactValidate_MissingTenant(t *testing.T) {
	a := testArtifact()
	a.TenantID = ""
	if err := a.Validate(); err == nil || !strings.Contains(err.Error(), "tenantId") {
		t.Errorf("expected tenantId error, got %v", err)
	}
}

func TestArtifactValidate_MissingEvidence(t *testing.T) {
	a := testArtifact()
	a.EvidenceID = ""
	if err := a.Validate(); err != ErrMissingEvidenceRef {
		t.Errorf("expected ErrMissingEvidenceRef, got %v", err)
	}
}

func TestArtifactValidate_InvalidType(t *testing.T) {
	a := testArtifact()
	a.ArtifactType = "badtype"
	if err := a.Validate(); err != ErrInvalidArtifactType {
		t.Errorf("expected ErrInvalidArtifactType, got %v", err)
	}
}

func TestArtifactValidate_InvalidVersion(t *testing.T) {
	a := testArtifact()
	a.Version = 0
	if err := a.Validate(); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("expected version error, got %v", err)
	}
}

func TestArtifactValidate_BadObjectKey(t *testing.T) {
	a := testArtifact()
	a.ObjectKey = "wrong/key"
	if err := a.Validate(); err != ErrInvalidNamingConvention {
		t.Errorf("expected ErrInvalidNamingConvention, got %v", err)
	}
}

func TestArtifactValidate_GoodObjectKey(t *testing.T) {
	a := testArtifact()
	a.ObjectKey = BuildObjectKey(a.TenantID, a.EvidenceID, a.ArtifactType, a.Version, a.Filename)
	if err := a.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpload_Success(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	data := []byte("test PDF content")

	meta, err := store.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if meta.ArtifactID == "" {
		t.Error("ArtifactID should be auto-generated")
	}
	if meta.WORMProtected != true {
		t.Error("WORMProtected should be true")
	}
	if meta.SizeBytes != int64(len(data)) {
		t.Errorf("SizeBytes = %d, want %d", meta.SizeBytes, len(data))
	}
	if meta.Checksum == "" {
		t.Error("Checksum should be computed")
	}
	if meta.ObjectKey != BuildObjectKey(a.TenantID, a.EvidenceID, a.ArtifactType, a.Version, a.Filename) {
		t.Errorf("ObjectKey mismatch: %s", meta.ObjectKey)
	}
}

func TestUpload_WORM_DuplicateRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	data := []byte("content")

	_, err := store.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("first Upload error: %v", err)
	}

	_, err = store.Upload(ctx, a, data)
	if err != ErrArtifactAlreadyExists {
		t.Errorf("expected ErrArtifactAlreadyExists, got %v", err)
	}
}

func TestUpload_WORM_FileIsReadOnly(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	data := []byte("WORM content")

	meta, err := store.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	fullPath := filepath.Join(store.rootDir, meta.ObjectKey)
	info, err := os.Stat(fullPath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.Mode().Perm() != 0444 {
		t.Errorf("file permission = %o, want 0444 (read-only)", info.Mode().Perm())
	}
}

func TestDownload_Success(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	data := []byte("downloadable content")

	meta, err := store.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	downloaded, dmeta, err := store.Download(ctx, a.TenantID, meta.ArtifactID)
	if err != nil {
		t.Fatalf("Download error: %v", err)
	}
	if string(downloaded) != string(data) {
		t.Error("downloaded data mismatch")
	}
	if dmeta.ArtifactID != meta.ArtifactID {
		t.Error("metadata mismatch")
	}
}

func TestDownload_NotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, _, err := store.Download(ctx, "tenant-001", "nonexistent-id")
	if err != ErrArtifactNotFound {
		t.Errorf("expected ErrArtifactNotFound, got %v", err)
	}
}

func TestDownload_CrossTenantForbidden(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	_, _, err = store.Download(ctx, "tenant-evil", meta.ArtifactID)
	if err != ErrCrossTenantAccess {
		t.Errorf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestDelete_WORM_Forbidden(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	err = store.Delete(ctx, a.TenantID, meta.ArtifactID)
	if err != ErrArtifactWORMDelete {
		t.Errorf("expected ErrArtifactWORMDelete, got %v", err)
	}
}

func TestGetMetadata_Success(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	got, err := store.GetMetadata(ctx, a.TenantID, meta.ArtifactID)
	if err != nil {
		t.Fatalf("GetMetadata error: %v", err)
	}
	if got.Checksum != meta.Checksum {
		t.Error("checksum mismatch")
	}
}

func TestGetMetadata_CrossTenantForbidden(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	_, err = store.GetMetadata(ctx, "tenant-evil", meta.ArtifactID)
	if err != ErrCrossTenantAccess {
		t.Errorf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestListByTenant(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	a1 := testArtifact()
	a2 := testArtifact()
	a2.EvidenceID = "evd-002"
	a2.Filename = "report2.pdf"

	store.Upload(ctx, a1, []byte("c1"))
	store.Upload(ctx, a2, []byte("c2"))

	list, err := store.ListByTenant(ctx, "tenant-001")
	if err != nil {
		t.Fatalf("ListByTenant error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByTenant len = %d, want 2", len(list))
	}

	list, _ = store.ListByTenant(ctx, "tenant-other")
	if len(list) != 0 {
		t.Errorf("ListByTenant for other tenant len = %d, want 0", len(list))
	}
}

func TestListByEvidence(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	a1 := testArtifact()
	a2 := testArtifact()
	a2.Version = 2

	store.Upload(ctx, a1, []byte("v1"))
	store.Upload(ctx, a2, []byte("v2"))

	list, err := store.ListByEvidence(ctx, "tenant-001", "evd-001")
	if err != nil {
		t.Fatalf("ListByEvidence error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByEvidence len = %d, want 2", len(list))
	}
}

func TestExists(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	exists, err := store.Exists(ctx, a.TenantID, meta.ArtifactID)
	if err != nil || !exists {
		t.Errorf("Exists = %v, err = %v, want true/nil", exists, err)
	}

	exists, err = store.Exists(ctx, a.TenantID, "nonexistent")
	if err != nil || exists {
		t.Errorf("Exists = %v, err = %v, want false/nil", exists, err)
	}
}

func TestExists_CrossTenantForbidden(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, _ := store.Upload(ctx, a, []byte("content"))

	_, err := store.Exists(ctx, "tenant-evil", meta.ArtifactID)
	if err != ErrCrossTenantAccess {
		t.Errorf("expected ErrCrossTenantAccess, got %v", err)
	}
}

func TestChecksum_Integrity(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	data := []byte("integrity check content")

	meta, err := store.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	expected := computeChecksum(data)
	if meta.Checksum != expected {
		t.Errorf("Checksum = %s, want %s", meta.Checksum, expected)
	}
}

func TestEvidenceTraceability(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()

	meta, err := store.Upload(ctx, a, []byte("content"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	if meta.EvidenceID != a.EvidenceID {
		t.Errorf("EvidenceID = %s, want %s", meta.EvidenceID, a.EvidenceID)
	}

	list, _ := store.ListByEvidence(ctx, a.TenantID, a.EvidenceID)
	for _, m := range list {
		if m.EvidenceID != a.EvidenceID {
			t.Errorf("artifact %s has EvidenceID %s, expected %s", m.ArtifactID, m.EvidenceID, a.EvidenceID)
		}
	}
}

func TestUpload_MissingEvidence_Rejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	a := testArtifact()
	a.EvidenceID = ""

	_, err := store.Upload(ctx, a, []byte("content"))
	if err != ErrMissingEvidenceRef {
		t.Errorf("expected ErrMissingEvidenceRef, got %v", err)
	}
}

func TestManifest_Persistence(t *testing.T) {
	dir := t.TempDir()

	store1, err := NewLocalArtifactStore(dir)
	if err != nil {
		t.Fatalf("NewLocalArtifactStore error: %v", err)
	}
	ctx := context.Background()
	a := testArtifact()
	data := []byte("persistent content")

	meta, err := store1.Upload(ctx, a, data)
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}

	store2, err := NewLocalArtifactStore(dir)
	if err != nil {
		t.Fatalf("NewLocalArtifactStore (reload) error: %v", err)
	}

	got, err := store2.GetMetadata(ctx, a.TenantID, meta.ArtifactID)
	if err != nil {
		t.Fatalf("GetMetadata after reload error: %v", err)
	}
	if got.Checksum != meta.Checksum {
		t.Error("checksum mismatch after manifest reload")
	}

	downloaded, _, err := store2.Download(ctx, a.TenantID, meta.ArtifactID)
	if err != nil {
		t.Fatalf("Download after reload error: %v", err)
	}
	if string(downloaded) != string(data) {
		t.Error("data mismatch after manifest reload")
	}
}

func newTestStore(t *testing.T) *LocalArtifactStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewLocalArtifactStore(dir)
	if err != nil {
		t.Fatalf("NewLocalArtifactStore error: %v", err)
	}
	return store
}
