package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/enterprise"
	"github.com/wujupeng/ebcx/internal/enterprise/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/enterprise/repository"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
)

type UpdateEnterpriseHandler struct {
	uow      repository.UnitOfWork
	repo     repository.EnterpriseRepository
	evWriter evidence_adapter.EnterpriseEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewUpdateEnterpriseHandler(
	uow repository.UnitOfWork,
	repo repository.EnterpriseRepository,
	evWriter evidence_adapter.EnterpriseEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *UpdateEnterpriseHandler {
	return &UpdateEnterpriseHandler{
		uow:      uow,
		repo:     repo,
		evWriter: evWriter,
		outbox:   obx,
		idemRepo: idemRepo,
	}
}

func (h *UpdateEnterpriseHandler) Handle(ctx context.Context, cmd enterprise.UpdateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "UpdateEnterprise", cmd.EnterpriseID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == enterprise.ErrIdempotencyRecordFailed {
			return h.retryUpdate(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &enterprise.EnterpriseAggregate{Version: firstResult.ResultVersion}, nil
	}

	existing, err := h.repo.FindByID(ctx, tx, cmd.EnterpriseID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	event, err := existing.UpdateEnterprise(cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	exists, err := h.repo.ExistsByName(ctx, tx, existing.TenantID, cmd.NewName, cmd.EnterpriseID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if exists {
		_ = h.uow.Rollback(tx)
		return nil, enterprise.ErrEnterpriseDuplicateName
	}

	agg, err := h.repo.UpdateWithCAS(ctx, tx, cmd.EnterpriseID, cmd.NewName, cmd.ExpectedVersion, cmd.SourceEvidenceID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, agg, event, "UpdateEnterprise")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"enterpriseId":     agg.EnterpriseID,
		"name":             agg.Name,
		"version":          agg.Version,
		"sourceEvidenceId": agg.SourceEvidenceID,
		"tenantId":         agg.TenantID,
		"timestamp":        event.GetTimestamp(),
		"traceId":          uuid.NewString(),
		"evidenceRef":      evRecord.EvidenceID,
	})

	if err := h.outbox.Write(ctx, tx, "Enterprise", agg.EnterpriseID,
		"enterprise.updated", agg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, enterprise.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, agg.TenantID,
		agg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, enterprise.ErrTransactionCommitFailed
	}

	return agg, nil
}

func (h *UpdateEnterpriseHandler) retryUpdate(ctx context.Context, cmd enterprise.UpdateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "UpdateEnterprise", cmd.EnterpriseID)
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &enterprise.EnterpriseAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}
