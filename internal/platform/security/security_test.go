package security

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testAuthGateway() *AuthGateway {
	return NewAuthGateway(
		[]byte("test-signing-key-at-least-32-bytes-long!!!"),
		"ebcx-issuer",
		"ebcx-audience",
		30*time.Minute,
	)
}

func TestJWT_IssueAndValidate(t *testing.T) {
	gw := testAuthGateway()
	token, err := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"manager"}, []string{"read", "write"})
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := gw.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if claims.TenantID != "tenant-001" {
		t.Errorf("TenantID = %s, want tenant-001", claims.TenantID)
	}
	if claims.UserID != "user-001" {
		t.Errorf("UserID = %s, want user-001", claims.UserID)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "manager" {
		t.Errorf("Roles = %v, want [manager]", claims.Roles)
	}
}

func TestJWT_MissingToken_Returns401(t *testing.T) {
	gw := testAuthGateway()
	_, err := gw.ValidateToken("")
	if err != ErrMissingToken {
		t.Errorf("expected ErrMissingToken, got %v", err)
	}
}

func TestJWT_InvalidToken_Returns401(t *testing.T) {
	gw := testAuthGateway()
	_, err := gw.ValidateToken("invalid.token.here")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWT_ExpiredToken_Returns401(t *testing.T) {
	gw := NewAuthGateway(
		[]byte("test-signing-key-at-least-32-bytes-long!!!"),
		"ebcx-issuer",
		"ebcx-audience",
		-1*time.Minute,
	)
	token, _ := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"user"}, []string{"read"})

	gw2 := testAuthGateway()
	_, err := gw2.ValidateToken(token)
	if err != ErrExpiredToken && err != ErrInvalidToken {
		t.Errorf("expected ErrExpiredToken or ErrInvalidToken, got %v", err)
	}
}

func TestJWT_RevokeToken(t *testing.T) {
	gw := testAuthGateway()
	token, _ := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"user"}, []string{"read"})

	_, err := gw.ValidateToken(token)
	if err != nil {
		t.Fatalf("first validate error: %v", err)
	}

	gw.RevokeToken(token)
	_, err = gw.ValidateToken(token)
	if err != ErrTokenRevoked {
		t.Errorf("expected ErrTokenRevoked, got %v", err)
	}
}

func TestJWT_InvalidIssuer(t *testing.T) {
	gw1 := NewAuthGateway([]byte("test-signing-key-at-least-32-bytes-long!!!"), "issuer-1", "ebcx-audience", 30*time.Minute)
	gw2 := NewAuthGateway([]byte("test-signing-key-at-least-32-bytes-long!!!"), "issuer-2", "ebcx-audience", 30*time.Minute)

	token, _ := gw1.IssueToken("tenant-001", "user-001", "org-001", []string{"user"}, []string{"read"})
	_, err := gw2.ValidateToken(token)
	if err != ErrInvalidIssuer {
		t.Errorf("expected ErrInvalidIssuer, got %v", err)
	}
}

func TestJWT_CheckScope(t *testing.T) {
	gw := testAuthGateway()
	token, _ := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"user"}, []string{"read", "write"})
	claims, _ := gw.ValidateToken(token)

	if err := gw.CheckScope(claims, "read"); err != nil {
		t.Errorf("CheckScope(read) error: %v", err)
	}
	if err := gw.CheckScope(claims, "admin"); err != ErrInsufficientScope {
		t.Errorf("expected ErrInsufficientScope, got %v", err)
	}
}

func TestJWT_CheckRole_AdminBypass(t *testing.T) {
	gw := testAuthGateway()
	token, _ := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"admin"}, []string{"read"})
	claims, _ := gw.ValidateToken(token)

	if err := gw.CheckRole(claims, "manager"); err != nil {
		t.Errorf("admin should bypass role check, got: %v", err)
	}
}

func TestJWT_ExtractBearerToken(t *testing.T) {
	token, err := ExtractBearerToken("Bearer abc.def.ghi")
	if err != nil {
		t.Fatalf("ExtractBearerToken error: %v", err)
	}
	if token != "abc.def.ghi" {
		t.Errorf("token = %s, want abc.def.ghi", token)
	}

	_, err = ExtractBearerToken("")
	if err != ErrMissingToken {
		t.Errorf("expected ErrMissingToken, got %v", err)
	}

	_, err = ExtractBearerToken("Basic abc")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWT_ValidationPerformance(t *testing.T) {
	gw := testAuthGateway()
	token, _ := gw.IssueToken("tenant-001", "user-001", "org-001", []string{"user"}, []string{"read"})

	iterations := 1000
	start := time.Now()
	for i := 0; i < iterations; i++ {
		gw.ValidateToken(token)
	}
	elapsed := time.Since(start)
	perOp := elapsed / time.Duration(iterations)

	if perOp > 10*time.Millisecond {
		t.Errorf("JWT validation avg %v > 10ms threshold", perOp)
	}
	t.Logf("JWT validation avg: %v (threshold: 10ms)", perOp)
}

func TestAudit_Append_Success(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	entry := AuditEntry{
		AuditID:    "audit-001",
		TenantID:   "tenant-001",
		UserID:     "user-001",
		Action:     "READ",
		Resource:   "order",
		ResourceID: "order-001",
		Decision:   "allow",
	}

	err := logger.Append(ctx, entry)
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if logger.Count() != 1 {
		t.Errorf("Count = %d, want 1", logger.Count())
	}
}

func TestAudit_Append_MissingTenant(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	entry := AuditEntry{
		Action:   "READ",
		TenantID: "",
	}

	err := logger.Append(ctx, entry)
	if err == nil {
		t.Error("expected error for missing tenantId")
	}
}

func TestAudit_Append_MissingAction(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	entry := AuditEntry{
		TenantID: "tenant-001",
		Action:   "",
	}

	err := logger.Append(ctx, entry)
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestAudit_Query_TenantIsolation(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	logger.Append(ctx, AuditEntry{AuditID: "a1", TenantID: "tenant-001", UserID: "u1", Action: "READ", Resource: "order", Decision: "allow"})
	logger.Append(ctx, AuditEntry{AuditID: "a2", TenantID: "tenant-002", UserID: "u2", Action: "READ", Resource: "order", Decision: "allow"})
	logger.Append(ctx, AuditEntry{AuditID: "a3", TenantID: "tenant-001", UserID: "u1", Action: "WRITE", Resource: "order", Decision: "deny"})

	results, err := logger.Query(ctx, "tenant-001", AuditQueryFilters{})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("tenant-001 results = %d, want 2", len(results))
	}

	results, _ = logger.Query(ctx, "tenant-002", AuditQueryFilters{})
	if len(results) != 1 {
		t.Errorf("tenant-002 results = %d, want 1", len(results))
	}
}

func TestAudit_Query_WithFilters(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	logger.Append(ctx, AuditEntry{AuditID: "a1", TenantID: "t1", UserID: "u1", Action: "READ", Resource: "order", Decision: "allow"})
	logger.Append(ctx, AuditEntry{AuditID: "a2", TenantID: "t1", UserID: "u1", Action: "WRITE", Resource: "order", Decision: "deny"})
	logger.Append(ctx, AuditEntry{AuditID: "a3", TenantID: "t1", UserID: "u2", Action: "READ", Resource: "order", Decision: "allow"})

	results, _ := logger.Query(ctx, "t1", AuditQueryFilters{UserID: "u1"})
	if len(results) != 2 {
		t.Errorf("filter by UserID=u1: %d, want 2", len(results))
	}

	results, _ = logger.Query(ctx, "t1", AuditQueryFilters{Action: "READ"})
	if len(results) != 2 {
		t.Errorf("filter by Action=READ: %d, want 2", len(results))
	}

	results, _ = logger.Query(ctx, "t1", AuditQueryFilters{UserID: "u1", Action: "WRITE"})
	if len(results) != 1 {
		t.Errorf("filter by UserID=u1+Action=WRITE: %d, want 1", len(results))
	}
}

func TestAudit_GetByID(t *testing.T) {
	logger := NewInMemoryAuditLogger()
	ctx := context.Background()

	logger.Append(ctx, AuditEntry{AuditID: "a1", TenantID: "t1", UserID: "u1", Action: "READ", Resource: "order", Decision: "allow"})

	entry, err := logger.GetByID(ctx, "t1", "a1")
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if entry.UserID != "u1" {
		t.Errorf("UserID = %s, want u1", entry.UserID)
	}

	_, err = logger.GetByID(ctx, "t1", "nonexistent")
	if err != ErrAuditNotFound {
		t.Errorf("expected ErrAuditNotFound, got %v", err)
	}

	_, err = logger.GetByID(ctx, "t-other", "a1")
	if err != ErrAuditNotFound {
		t.Errorf("expected ErrAuditNotFound for cross-tenant, got %v", err)
	}
}

func TestAudit_EntryFromClaims(t *testing.T) {
	claims := &Claims{
		TenantID: "tenant-001",
		UserID:   "user-001",
	}
	entry := AuditEntryFromClaims(claims, "READ", "order", "order-001", "allow", "policy-match", "10.0.0.1", "test-agent")

	if entry.TenantID != "tenant-001" || entry.UserID != "user-001" {
		t.Error("AuditEntryFromClaims mismatch")
	}
	if entry.Action != "READ" || entry.Resource != "order" {
		t.Error("AuditEntryFromClaims action/resource mismatch")
	}
}

func TestKMS_GenerateAndEncrypt(t *testing.T) {
	kms := NewKMS()
	err := kms.GenerateKey("key-001")
	if err != nil {
		t.Fatalf("GenerateKey error: %v", err)
	}

	plaintext := "sensitive-data-123"
	encrypted, err := kms.EncryptField(plaintext)
	if err != nil {
		t.Fatalf("EncryptField error: %v", err)
	}
	if encrypted == plaintext {
		t.Error("encrypted should differ from plaintext")
	}
	if !strings.Contains(encrypted, "key-001:") {
		t.Errorf("encrypted should contain key ID prefix: %s", encrypted)
	}
}

func TestKMS_Decrypt_RoundTrip(t *testing.T) {
	kms := NewKMS()
	kms.GenerateKey("key-001")

	plaintext := "secret-value"
	encrypted, _ := kms.EncryptField(plaintext)

	decrypted, err := kms.DecryptField(encrypted)
	if err != nil {
		t.Fatalf("DecryptField error: %v", err)
	}
	if decrypted != plaintext {
		t.Errorf("decrypted = %s, want %s", decrypted, plaintext)
	}
}

func TestKMS_KeyRotation(t *testing.T) {
	kms := NewKMS()
	kms.GenerateKey("key-001")

	encrypted, _ := kms.EncryptField("old-key-data")

	err := kms.RotateKey("key-002")
	if err != nil {
		t.Fatalf("RotateKey error: %v", err)
	}
	if kms.ActiveKeyID() != "key-002" {
		t.Errorf("active key = %s, want key-002", kms.ActiveKeyID())
	}

	decrypted, err := kms.DecryptField(encrypted)
	if err != nil {
		t.Fatalf("decrypt with old key should still work: %v", err)
	}
	if decrypted != "old-key-data" {
		t.Errorf("decrypted = %s, want old-key-data", decrypted)
	}

	newEncrypted, _ := kms.EncryptField("new-key-data")
	if !strings.Contains(newEncrypted, "key-002:") {
		t.Error("new encryption should use key-002")
	}
}

func TestKMS_NoKey_Error(t *testing.T) {
	kms := NewKMS()
	_, err := kms.EncryptField("data")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestKMS_InvalidCiphertext(t *testing.T) {
	kms := NewKMS()
	kms.GenerateKey("key-001")

	_, err := kms.Decrypt("invalid-format")
	if err != ErrInvalidCiphertext {
		t.Errorf("expected ErrInvalidCiphertext, got %v", err)
	}

	_, err = kms.Decrypt("nonexistent-key:deadbeef")
	if err != ErrKeyNotFound {
		t.Errorf("expected ErrKeyNotFound, got %v", err)
	}
}

func TestKMS_HashPII(t *testing.T) {
	kms := NewKMS()
	h1 := kms.HashPII("user@example.com")
	h2 := kms.HashPII("user@example.com")
	h3 := kms.HashPII("other@example.com")

	if h1 != h2 {
		t.Error("same input should produce same hash")
	}
	if h1 == h3 {
		t.Error("different input should produce different hash")
	}
	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 (SHA-256)", len(h1))
	}
}

func TestTLS_RejectLowVersion(t *testing.T) {
	gw := NewTLSGateway()
	err := gw.CheckTLSVersion(tls.VersionTLS10)
	if !errors.Is(err, ErrTLSVersionTooLow) {
		t.Errorf("expected ErrTLSVersionTooLow, got %v", err)
	}
}

func TestTLS_AcceptTLS12(t *testing.T) {
	gw := NewTLSGateway()
	err := gw.CheckTLSVersion(tls.VersionTLS12)
	if err != nil {
		t.Errorf("TLS 1.2 should be accepted, got: %v", err)
	}
}

func TestTLS_AcceptTLS13(t *testing.T) {
	gw := NewTLSGateway()
	err := gw.CheckTLSVersion(tls.VersionTLS13)
	if err != nil {
		t.Errorf("TLS 1.3 should be accepted, got: %v", err)
	}
}

func TestTLS_RejectWeakCipher(t *testing.T) {
	gw := NewTLSGateway()
	err := gw.CheckCipherSuite(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA)
	if !errors.Is(err, ErrTLSWeakCipher) {
		t.Errorf("expected ErrTLSWeakCipher, got %v", err)
	}
}

func TestTLS_AcceptStrongCipher(t *testing.T) {
	gw := NewTLSGateway()
	err := gw.CheckCipherSuite(tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384)
	if err != nil {
		t.Errorf("strong cipher should be accepted, got: %v", err)
	}
}

func TestTLS_ConfigMinVersion(t *testing.T) {
	gw := NewTLSGateway()
	config := gw.TLSConfig()
	if config.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want %d", config.MinVersion, tls.VersionTLS12)
	}
	if len(config.CipherSuites) == 0 {
		t.Error("should have allowed cipher suites")
	}
}

func TestSecurityLint_NoViolations(t *testing.T) {
	violations, err := LintModuleSecurity("../../..")
	if err != nil {
		t.Fatalf("LintModuleSecurity error: %v", err)
	}
	for _, v := range violations {
		t.Errorf("security lint violation: %s - %s (import: %s)", v.File, v.Violation, v.ImportPath)
	}
}

func TestClaims_Structure(t *testing.T) {
	claims := &Claims{
		TenantID: "t1",
		UserID:   "u1",
		OrgID:    "o1",
		Roles:    []string{"admin"},
		Scopes:   []string{"read", "write"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  "test",
			Subject: "u1",
		},
	}

	if claims.TenantID != "t1" || claims.UserID != "u1" || claims.OrgID != "o1" {
		t.Error("Claims fields mismatch")
	}
	if len(claims.Roles) != 1 || len(claims.Scopes) != 2 {
		t.Error("Claims Roles/Scopes length mismatch")
	}
}
