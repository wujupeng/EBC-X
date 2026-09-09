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

type CreateEnterpriseHandler struct {
	uow      repository.UnitOfWork
	repo     repository.EnterpriseRepository
	evWriter evidence_adapter.EnterpriseEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewCreateEnterpriseHandler(
	uow repository.UnitOfWork,
	repo repository.EnterpriseRepository,
	evWriter evidence_adapter.EnterpriseEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *CreateEnterpriseHandler {
	return &CreateEnterpriseHandler{
		uow:      uow,
		repo:     repo,
		evWriter: evWriter,
		outbox:   obx,
		idemRepo: idemRepo,
	}
}

func (h *CreateEnterpriseHandler) Handle(ctx context.Context, cmd enterprise.CreateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreateEnterprise", "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == enterprise.ErrIdempotencyRecordFailed {
			return h.retryCreate(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &enterprise.EnterpriseAggregate{Version: firstResult.ResultVersion}, nil
	}

	agg := enterprise.NewEnterpriseAggregate()
	event, err := agg.CreateEnterprise(cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	exists, err := h.repo.ExistsByName(ctx, tx, cmd.TenantID, cmd.Name, "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if exists {
		_ = h.uow.Rollback(tx)
		return nil, enterprise.ErrEnterpriseDuplicateName
	}

	if err := h.repo.Insert(ctx, tx, agg); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, agg, event, "CreateEnterprise")
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
		"enterprise.created", agg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, enterprise.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, cmd.TenantID,
		agg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, enterprise.ErrTransactionCommitFailed
	}

	return agg, nil
}

func (h *CreateEnterpriseHandler) retryCreate(ctx context.Context, cmd enterprise.CreateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreateEnterprise", "")
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &enterprise.EnterpriseAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}
