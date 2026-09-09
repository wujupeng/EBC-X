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

type CreateOrganizationHandler struct {
	uow      repository.UnitOfWork
	repo     repository.OrganizationRepository
	evWriter evidence_adapter.OrganizationEvidenceWriter
	outbox   *outbox.Publisher
	idemRepo repository.CommandIdempotencyRepository
}

func NewCreateOrganizationHandler(
	uow repository.UnitOfWork,
	repo repository.OrganizationRepository,
	evWriter evidence_adapter.OrganizationEvidenceWriter,
	obx *outbox.Publisher,
	idemRepo repository.CommandIdempotencyRepository,
) *CreateOrganizationHandler {
	return &CreateOrganizationHandler{uow: uow, repo: repo, evWriter: evWriter, outbox: obx, idemRepo: idemRepo}
}

func (h *CreateOrganizationHandler) Handle(ctx context.Context, cmd organization.CreateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreateOrganization", "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == organization.ErrIdempotencyRecordFailed {
			return h.retryCreate(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}

	exists, err := h.repo.CheckEnterpriseExists(ctx, tx, cmd.EnterpriseID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if !exists {
		_ = h.uow.Rollback(tx)
		return nil, organization.ErrOrgEnterpriseNotFound
	}

	parentLevel := 0
	if cmd.ParentID != "" {
		parent, err := h.repo.FindByID(ctx, tx, cmd.ParentID, false)
		if err != nil {
			_ = h.uow.Rollback(tx)
			return nil, err
		}
		if parent.EnterpriseID != cmd.EnterpriseID {
			_ = h.uow.Rollback(tx)
			return nil, organization.ErrOrgParentCrossEnterprise
		}
		parentLevel = parent.Level
	}

	agg := organization.NewOrganizationAggregate()
	event, err := agg.CreateOrganization(cmd, parentLevel)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	codeExists, err := h.repo.ExistsByCode(ctx, tx, cmd.EnterpriseID, cmd.ParentID, cmd.Code, "")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}
	if codeExists {
		_ = h.uow.Rollback(tx)
		return nil, organization.ErrOrgDuplicateCode
	}

	if err := h.repo.Insert(ctx, tx, agg); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	evRecord, err := h.evWriter.Write(ctx, tx, agg, event, "CreateOrganization")
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	payload, _ := json.Marshal(map[string]any{
		"orgId":            agg.OrgID,
		"enterpriseId":     agg.EnterpriseID,
		"parentId":         agg.ParentID,
		"name":             agg.Name,
		"code":             agg.Code,
		"level":            agg.Level,
		"version":          agg.Version,
		"sourceEvidenceId": agg.SourceEvidenceID,
		"tenantId":         agg.TenantID,
		"timestamp":        event.GetTimestamp(),
		"traceId":          uuid.NewString(),
		"evidenceRef":      evRecord.EvidenceID,
	})

	if err := h.outbox.Write(ctx, tx, "Organization", agg.OrgID, "organization.created", agg.TenantID, "", cmd.SourceEvidenceID, payload); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, organization.ErrOutboxWriteFailed
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, cmd.TenantID, agg.Version, event.GetEventID(), evRecord.EvidenceID); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, organization.ErrTransactionCommitFailed
	}

	return agg, nil
}

func (h *CreateOrganizationHandler) retryCreate(ctx context.Context, cmd organization.CreateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	tx, err := h.uow.BeginTenantTx(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, cmd.TenantID, "CreateOrganization", "")
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}