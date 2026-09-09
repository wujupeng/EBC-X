package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/organization"
	"github.com/wujupeng/ebcx/internal/organization/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/organization/repository"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
)

type UpdateOrganizationHandler struct {
	uow      repository.UnitOfWork
	repo     repository.OrganizationRepository
	evWriter evidence_adapter.OrganizationEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewUpdateOrganizationHandler(
	uow repository.UnitOfWork,
	repo repository.OrganizationRepository,
	evWriter evidence_adapter.OrganizationEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *UpdateOrganizationHandler {
	return &UpdateOrganizationHandler{uow: uow, repo: repo, evWriter: evWriter, outbox: obx, idemRepo: idemRepo}
}

func (h *UpdateOrganizationHandler) Handle(ctx context.Context, cmd organization.UpdateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, "", "UpdateOrganization", cmd.OrgID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == organization.ErrIdempotencyRecordFailed {
			return h.retryUpdate(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}

	existing, err := h.repo.FindByID(ctx, tx, cmd.OrgID, true)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	event, err := existing.UpdateOrganization(cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	codeExists, err := h.repo.ExistsByCode(ctx, tx, existing.EnterpriseID, existing.ParentID, cmd.NewCode, cmd.OrgID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if codeExists {
		_ = h.uow.Rollback(tx)
		return nil, organization.ErrOrgDuplicateCode
	}

	agg, err := h.repo.UpdateWithCAS(ctx, tx, cmd.OrgID, cmd.NewName, cmd.NewCode, cmd.ExpectedVersion, cmd.SourceEvidenceID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, agg, event, "UpdateOrganization")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"orgId":            agg.OrgID,
		"enterpriseId":     agg.EnterpriseID,
		"newName":          agg.Name,
		"newCode":          agg.Code,
		"version":          agg.Version,
		"sourceEvidenceId": agg.SourceEvidenceID,
		"tenantId":         agg.TenantID,
		"timestamp":        event.GetTimestamp(),
		"traceId":          uuid.NewString(),
		"evidenceRef":      evRecord.EvidenceID,
	})

	if err := h.outbox.Write(ctx, tx, "Organization", agg.OrgID, "organization.updated", agg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, organization.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, agg.TenantID, agg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, organization.ErrTransactionCommitFailed
	}

	return agg, nil
}

func (h *UpdateOrganizationHandler) retryUpdate(ctx context.Context, cmd organization.UpdateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, "")
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, "", "UpdateOrganization", cmd.OrgID)
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}