package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ArtifactStore interface {
	Upload(ctx context.Context, artifact Artifact, data []byte) (*ArtifactMetadata, error)
	Download(ctx context.Context, tenantID, artifactID string) ([]byte, *ArtifactMetadata, error)
	Delete(ctx context.Context, tenantID, artifactID string) error
	GetMetadata(ctx context.Context, tenantID, artifactID string) (*ArtifactMetadata, error)
	ListByTenant(ctx context.Context, tenantID string) ([]ArtifactMetadata, error)
	ListByEvidence(ctx context.Context, tenantID, evidenceID string) ([]ArtifactMetadata, error)
	Exists(ctx context.Context, tenantID, artifactID string) (bool, error)
}

type LocalArtifactStore struct {
	rootDir  string
	mu       sync.RWMutex
	metadata map[string]*ArtifactMetadata
	wormFile string
}

func NewLocalArtifactStore(rootDir string) (*LocalArtifactStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact root dir: %w", err)
	}
	store := &LocalArtifactStore{
		rootDir:  rootDir,
		metadata: make(map[string]*ArtifactMetadata),
		wormFile: filepath.Join(rootDir, ".worm_manifest"),
	}
	store.loadManifest()
	return store, nil
}

func (s *LocalArtifactStore) Upload(ctx context.Context, artifact Artifact, data []byte) (*ArtifactMetadata, error) {
	if err := artifact.Validate(); err != nil {
		return nil, err
	}

	artifactID := artifact.ArtifactID
	if artifactID == "" {
		artifactID = generateArtifactID(artifact)
		artifact.ArtifactID = artifactID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.metadata[artifactID]; exists {
		return nil, ErrArtifactAlreadyExists
	}

	objectKey := artifact.ObjectKeyCanonical()
	fullPath := filepath.Join(s.rootDir, objectKey)

	if _, err := os.Stat(fullPath); err == nil {
		return nil, ErrArtifactAlreadyExists
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create artifact directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0444); err != nil {
		return nil, fmt.Errorf("failed to write artifact (WORM): %w", err)
	}

	checksum := computeChecksum(data)
	now := time.Now()

	metadata := &ArtifactMetadata{
		ArtifactID:    artifactID,
		TenantID:      artifact.TenantID,
		EvidenceID:    artifact.EvidenceID,
		ArtifactType:  artifact.ArtifactType,
		Version:       artifact.Version,
		Filename:      artifact.Filename,
		ContentType:   artifact.ContentType,
		SizeBytes:     int64(len(data)),
		Checksum:      checksum,
		UploadedAt:    now,
		UploadedBy:    artifact.UploadedBy,
		ObjectKey:     objectKey,
		WORMProtected: true,
	}

	s.metadata[artifactID] = metadata
	s.saveManifest()

	return metadata, nil
}

func (s *LocalArtifactStore) Download(ctx context.Context, tenantID, artifactID string) ([]byte, *ArtifactMetadata, error) {
	s.mu.RLock()
	meta, exists := s.metadata[artifactID]
	s.mu.RUnlock()

	if !exists {
		return nil, nil, ErrArtifactNotFound
	}

	if meta.TenantID != tenantID {
		return nil, nil, ErrCrossTenantAccess
	}

	fullPath := filepath.Join(s.rootDir, meta.ObjectKey)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read artifact: %w", err)
	}

	return data, meta, nil
}

func (s *LocalArtifactStore) Delete(ctx context.Context, tenantID, artifactID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, exists := s.metadata[artifactID]
	if !exists {
		return ErrArtifactNotFound
	}

	if meta.TenantID != tenantID {
		return ErrCrossTenantAccess
	}

	return ErrArtifactWORMDelete
}

func (s *LocalArtifactStore) GetMetadata(ctx context.Context, tenantID, artifactID string) (*ArtifactMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, exists := s.metadata[artifactID]
	if !exists {
		return nil, ErrArtifactNotFound
	}

	if meta.TenantID != tenantID {
		return nil, ErrCrossTenantAccess
	}

	return meta, nil
}

func (s *LocalArtifactStore) ListByTenant(ctx context.Context, tenantID string) ([]ArtifactMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ArtifactMetadata, 0)
	for _, meta := range s.metadata {
		if meta.TenantID == tenantID {
			result = append(result, *meta)
		}
	}
	return result, nil
}

func (s *LocalArtifactStore) ListByEvidence(ctx context.Context, tenantID, evidenceID string) ([]ArtifactMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]ArtifactMetadata, 0)
	for _, meta := range s.metadata {
		if meta.TenantID == tenantID && meta.EvidenceID == evidenceID {
			result = append(result, *meta)
		}
	}
	return result, nil
}

func (s *LocalArtifactStore) Exists(ctx context.Context, tenantID, artifactID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, exists := s.metadata[artifactID]
	if !exists {
		return false, nil
	}
	if meta.TenantID != tenantID {
		return false, ErrCrossTenantAccess
	}
	return true, nil
}

func (s *LocalArtifactStore) loadManifest() {
	data, err := os.ReadFile(s.wormFile)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 10 {
			continue
		}
		meta := &ArtifactMetadata{
			ArtifactID:    parts[0],
			TenantID:      parts[1],
			EvidenceID:    parts[2],
			ArtifactType:  ArtifactType(parts[3]),
			Filename:      parts[4],
			ContentType:   parts[5],
			ObjectKey:     parts[6],
			Checksum:      parts[7],
			WORMProtected: true,
		}
		fmt.Sscanf(parts[8], "%d", &meta.Version)
		fmt.Sscanf(parts[9], "%d", &meta.SizeBytes)
		if len(parts) > 10 {
			meta.UploadedBy = parts[10]
		}
		s.metadata[meta.ArtifactID] = meta
	}
}

func (s *LocalArtifactStore) saveManifest() {
	var lines []string
	for _, meta := range s.metadata {
		line := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%d|%d|%s",
			meta.ArtifactID, meta.TenantID, meta.EvidenceID,
			string(meta.ArtifactType), meta.Filename, meta.ContentType,
			meta.ObjectKey, meta.Checksum, meta.Version, meta.SizeBytes, meta.UploadedBy)
		lines = append(lines, line)
	}
	data := strings.Join(lines, "\n")
	os.WriteFile(s.wormFile, []byte(data), 0644)
}

func computeChecksum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func generateArtifactID(a Artifact) string {
	h := sha256.New()
	h.Write([]byte(a.ObjectKeyCanonical()))
	return hex.EncodeToString(h.Sum(nil))[:36]
}
