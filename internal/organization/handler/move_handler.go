package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/wujupeng/ebcx/internal/organization"
	"github.com/wujupeng/ebcx/internal/organization/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/organization/repository"
)

type MoveOrganizationHandler struct {
	uow           repository.UnitOfWork
	repo          repository.OrganizationRepository
	evWriter      evidence_adapter.OrganizationEvidenceWriter
	structEvWriter evidence_adapter.OrganizationTreeStructuralEvidenceWriter
	idemRepo      repository.CommandIdempotencyRepository
	coordinator   *organization.OrganizationTreeCoordinator
}

func NewMoveOrganizationHandler(
	uow repository.UnitOfWork,
	repo repository.OrganizationRepository,
	evWriter evidence_adapter.OrganizationEvidenceWriter,
	structEvWriter evidence_adapter.OrganizationTreeStructuralEvidenceWriter,
	idemRepo repository.CommandIdempotencyRepository,
	coordinator *organization.OrganizationTreeCoordinator,
) *MoveOrganizationHandler {
	return &MoveOrganizationHandler{
		uow:            uow,
		repo:           repo,
		evWriter:       evWriter,
		structEvWriter: structEvWriter,
		idemRepo:       idemRepo,
		coordinator:    coordinator,
	}
}

func (h *MoveOrganizationHandler) Handle(ctx context.Context, cmd organization.MoveOrganizationCommand) (*organization.OrganizationAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	existing, err := h.repo.FindByID(ctx, nil, cmd.OrgID, false)
	if err != nil {
		return nil, err
	}

	tx, err := h.uow.BeginTenantTx(ctx, existing.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tenant tx: %w", err)
	}

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, existing.TenantID, "MoveOrganization", cmd.OrgID)
	if err != nil {
		_ = h.uow.Rollback(tx)
		if err == organization.ErrIdempotencyRecordFailed {
			return h.retryMove(ctx, cmd)
		}
		return nil, err
	}
	if firstResult != nil {
		_ = h.uow.Rollback(tx)
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}

	event, subtreeResult, err := h.coordinator.MoveSubtree(ctx, tx, cmd)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	_ = subtreeResult

	movedAgg, err := h.repo.FindByID(ctx, tx, cmd.OrgID, false)
	if err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.idemRepo.MarkSuccess(ctx, tx, cmd.CommandID, existing.TenantID, event.Version, event.EventID, ""); err != nil {
		_ = h.uow.Rollback(tx)
		return nil, err
	}

	if err := h.uow.Commit(tx); err != nil {
		return nil, organization.ErrTransactionCommitFailed
	}

	return movedAgg, nil
}

func (h *MoveOrganizationHandler) retryMove(ctx context.Context, cmd organization.MoveOrganizationCommand) (*organization.OrganizationAggregate, error) {
	existing, err := h.repo.FindByID(ctx, nil, cmd.OrgID, false)
	if err != nil {
		return nil, err
	}

	tx, err := h.uow.BeginTenantTx(ctx, existing.TenantID)
	if err != nil {
		return nil, err
	}
	defer h.uow.Rollback(tx)

	firstResult, err := h.idemRepo.CheckAndReserve(ctx, tx, cmd.CommandID, existing.TenantID, "MoveOrganization", cmd.OrgID)
	if err != nil {
		return nil, err
	}
	if firstResult != nil {
		return &organization.OrganizationAggregate{Version: firstResult.ResultVersion}, nil
	}
	return nil, fmt.Errorf("idempotency retry failed unexpectedly")
}