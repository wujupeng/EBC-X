package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/person"
	"github.com/wujupeng/ebcx/internal/person/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/person/repository"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
)

type CreatePersonHandler struct {
	uow      repository.UnitOfWork
	repo     repository.PersonRepository
	evWriter evidence_adapter.PersonEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewCreatePersonHandler(
	uow repository.UnitOfWork,
	repo repository.PersonRepository,
	evWriter evidence_adapter.PersonEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *CreatePersonHandler {
	return &CreatePersonHandler{uow: uow, repo: repo, evWriter: evWriter, outbox: obx, idemRepo: idemRepo}
}

func (h *CreatePersonHandler) Handle(ctx context.Context, cmd person.CreatePersonCommand) (*person.PersonAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreatePerson", "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == person.ErrIdempotencyRecordFailed {
			return h.retryCreate(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &person.PersonAggregate{Version: firstResult.ResultVersion}, nil
	}

	orgExists, err := h.repo.CheckOrganizationExists(ctx, tx, cmd.OrgID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if !orgExists {
		_ = h.uow.Rollback(tx)
		return nil, person.ErrPersonOrganizationNotFound
	}

	empExists, err := h.repo.ExistsByEmployeeNo(ctx, tx, cmd.OrgID, cmd.EmployeeNo, "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if empExists {
		_ = h.uow.Rollback(tx)
		return nil, person.ErrPersonDuplicateEmployeeNo
	}

	agg := person.NewPersonAggregate()
	event, err := agg.CreatePerson(cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.repo.Insert(ctx, tx, agg); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, agg, event, "CREATE")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"personId":         agg.PersonID,
		"orgId":            agg.OrgID,
		"name":             agg.Name,
		"employeeNo":       agg.EmployeeNo,
		"roles":            agg.Roles,
		"version":          agg.Version,
		"sourceEvidenceId": agg.SourceEvidenceID,
		"tenantId":         agg.TenantID,
		"timestamp":        event.GetTimestamp(),
		"traceId":          uuid.NewString(),
		"evidenceRef":      evRecord.EvidenceID,
	})

	if err := h.outbox.Write(ctx, tx, "Person", agg.PersonID, "person.created", agg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, person.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, cmd.TenantID, agg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, person.ErrTransactionCommitFailed
	}

	return agg, nil
}

func (h *CreatePersonHandler) retryCreate(ctx context.Context, cmd person.CreatePersonCommand) (*person.PersonAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreatePerson", "")
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &person.PersonAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}
