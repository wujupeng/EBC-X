package handler

import (
	"context"

	"github.com/wujupeng/ebcx/internal/organization"
)

type OrganizationCommandBus struct {
	createHandler *CreateOrganizationHandler
	updateHandler *UpdateOrganizationHandler
	moveHandler   *MoveOrganizationHandler
}

func NewOrganizationCommandBus(
	createHandler *CreateOrganizationHandler,
	updateHandler *UpdateOrganizationHandler,
	moveHandler *MoveOrganizationHandler,
) *OrganizationCommandBus {
	return &OrganizationCommandBus{
		createHandler: createHandler,
		updateHandler: updateHandler,
		moveHandler:   moveHandler,
	}
}

func (b *OrganizationCommandBus) DispatchCreate(ctx context.Context, cmd organization.CreateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	return b.createHandler.Handle(ctx, cmd)
}

func (b *OrganizationCommandBus) DispatchUpdate(ctx context.Context, cmd organization.UpdateOrganizationCommand) (*organization.OrganizationAggregate, error) {
	return b.updateHandler.Handle(ctx, cmd)
}

func (b *OrganizationCommandBus) DispatchMove(ctx context.Context, cmd organization.MoveOrganizationCommand) (*organization.OrganizationAggregate, error) {
	return b.moveHandler.Handle(ctx, cmd)
}
