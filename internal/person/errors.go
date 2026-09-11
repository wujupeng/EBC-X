package person

import "errors"

var (
	ErrPersonNotFound               = errors.New("EBCX-PERSON-NOT-FOUND")
	ErrPersonDuplicateEmployeeNo    = errors.New("EBCX-PERSON-DUPLICATE-EMPLOYEE-NO")
	ErrPersonOrganizationNotFound   = errors.New("EBCX-PERSON-ORGANIZATION-NOT-FOUND")
	ErrPersonOrgIDImmutable         = errors.New("EBCX-PERSON-ORGID-IMMUTABLE")
	ErrPersonRolesImmutableInUpdate = errors.New("EBCX-PERSON-ROLES-IMMUTABLE-IN-UPDATE")
	ErrPersonInvalidRoleRef         = errors.New("EBCX-PERSON-INVALID-ROLE-REF")
	ErrPersonRoleAlreadyAssigned    = errors.New("EBCX-PERSON-ROLE-ALREADY-ASSIGNED")
	ErrPersonRemoveRoleNotSupported = errors.New("EBCX-PERSON-REMOVE-ROLE-NOT-SUPPORTED")
	ErrPersonCASConflict            = errors.New("EBCX-PERSON-CAS-CONCURRENCY-CONFLICT")
	ErrPersonVersionMonotonic       = errors.New("EBCX-PERSON-VERSION-MONOTONIC-VIOLATION")
	ErrPersonInvalidName            = errors.New("EBCX-PERSON-INVALID-NAME")
	ErrPersonInvalidEmployeeNo      = errors.New("EBCX-PERSON-INVALID-EMPLOYEE-NO")
	ErrPersonInvalidCommandID       = errors.New("EBCX-PERSON-INVALID-COMMAND-ID")
	ErrPersonCommandIDRequired      = errors.New("EBCX-PERSON-COMMAND-ID-REQUIRED")
	ErrPersonIdempotencyConflict    = errors.New("EBCX-PERSON-IDEMPOTENCY-CONFLICT")
	ErrPersonUnknownCommand         = errors.New("EBCX-PERSON-UNKNOWN-COMMAND")
	ErrTenantContextMismatch        = errors.New("EBCX-TENANT-CONTEXT-MISMATCH")
	ErrTenantAccessDenied           = errors.New("EBCX-TENANT-ACCESS-DENIED")
	ErrEvidenceWriteFailed          = errors.New("EBCX-EVIDENCE-WRITE-FAILED")
	ErrOutboxWriteFailed            = errors.New("EBCX-OUTBOX-WRITE-FAILED")
	ErrTransactionCommitFailed      = errors.New("EBCX-TRANSACTION-COMMIT-FAILED")
	ErrIdempotencyRecordFailed      = errors.New("EBCX-IDEMPOTENCY-RECORD-FAILED")
)
