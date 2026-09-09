package enterprise

import "errors"

var (
	ErrEnterpriseDuplicateName       = errors.New("EBCX-ENTERPRISE-DUPLICATE-NAME")
	ErrEnterpriseNotFound            = errors.New("EBCX-ENTERPRISE-NOT-FOUND")
	ErrEnterpriseVersionConflict     = errors.New("EBCX-ENTERPRISE-VERSION-CONFLICT")
	ErrEnterpriseVersionMonotonic    = errors.New("EBCX-ENTERPRISE-VERSION-MONOTONIC-VIOLATION")
	ErrEnterpriseInvalidName         = errors.New("EBCX-ENTERPRISE-INVALID-NAME")
	ErrEnterpriseInvalidCommandID    = errors.New("EBCX-ENTERPRISE-INVALID-COMMAND-ID")
	ErrEnterpriseCommandIDRequired   = errors.New("EBCX-ENTERPRISE-COMMAND-ID-REQUIRED")
	ErrEnterpriseIdempotencyConflict = errors.New("EBCX-ENTERPRISE-IDEMPOTENCY-CONFLICT")
	ErrTenantContextMismatch         = errors.New("EBCX-TENANT-CONTEXT-MISMATCH")
	ErrTenantAccessDenied            = errors.New("EBCX-TENANT-ACCESS-DENIED")
	ErrEvidenceWriteFailed           = errors.New("EBCX-EVIDENCE-WRITE-FAILED")
	ErrOutboxWriteFailed             = errors.New("EBCX-OUTBOX-WRITE-FAILED")
	ErrTransactionCommitFailed       = errors.New("EBCX-TRANSACTION-COMMIT-FAILED")
	ErrIdempotencyRecordFailed       = errors.New("EBCX-IDEMPOTENCY-RECORD-FAILED")
	ErrUnknownCommand                = errors.New("EBCX-ENTERPRISE-UNKNOWN-COMMAND")
)
