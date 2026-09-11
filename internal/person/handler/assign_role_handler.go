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

type AssignRoleHandler struct {
	uow      repository.UnitOfWork
	repo     repository.PersonRepository
	evWriter evidence_adapter.PersonEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewAssignRoleHandler(
	uow repository.UnitOfWork,
	repo repository.PersonRepository,
	evWriter evidence_adapter.PersonEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *AssignRoleHandler {
	return &AssignRoleHandler{uow: uow, repo: repo, evWriter: evWriter, outbox: obx, idemRepo: idemRepo}
}

func (h *AssignRoleHandler) Handle(ctx context.Context, cmd person.AssignRoleCommand) (*person.PersonAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "AssignRole", cmd.PersonID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == person.ErrIdempotencyRecordFailed {
			return h.retryAssignRole(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &person.PersonAggregate{Version: firstResult.ResultVersion}, nil
	}

	existing, err := h.repo.FindByID(ctx, tx, cmd.PersonID, true)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	agg := existing
	event, err := agg.AssignRole(cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	updatedAgg, err := h.repo.AssignRoleWithCAS(ctx, tx, cmd.PersonID, cmd.RoleID, cmd.ExpectedVersion, cmd.SourceEvidenceID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, updatedAgg, event, "ASSIGN_ROLE")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"personId":         updatedAgg.PersonID,
		"roleId":           cmd.RoleID,
		"version":          updatedAgg.Version,
		"sourceEvidenceId": updatedAgg.SourceEvidenceID,
		"tenantId":         updatedAgg.TenantID,
		"timestamp":        event.GetTimestamp(),
		"traceId":          uuid.NewString(),
		"evidenceRef":      evRecord.EvidenceID,
	})

	if err := h.outbox.Write(ctx, tx, "Person", updatedAgg.PersonID, "role.assigned", updatedAgg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, person.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, cmd.TenantID, updatedAgg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, person.ErrTransactionCommitFailed
	}

	return updatedAgg, nil
}

func (h *AssignRoleHandler) retryAssignRole(ctx context.Context, cmd person.AssignRoleCommand) (*person.PersonAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "AssignRole", cmd.PersonID)
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &person.PersonAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}
