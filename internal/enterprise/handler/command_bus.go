package handler

import (
	"context"

	"github.com/wujupeng/ebcx/internal/enterprise"
)

type EnterpriseCommandBus struct {
	createHandler *CreateEnterpriseHandler
	updateHandler *UpdateEnterpriseHandler
}

func NewEnterpriseCommandBus(
	createHandler *CreateEnterpriseHandler,
	updateHandler *UpdateEnterpriseHandler,
) *EnterpriseCommandBus {
	return &EnterpriseCommandBus{
		createHandler: createHandler,
		updateHandler: updateHandler,
	}
}

func (b *EnterpriseCommandBus) DispatchCreate(ctx context.Context, cmd enterprise.CreateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	return b.createHandler.Handle(ctx, cmd)
}

func (b *EnterpriseCommandBus) DispatchUpdate(ctx context.Context, cmd enterprise.UpdateEnterpriseCommand) (*enterprise.EnterpriseAggregate, error) {
	return b.updateHandler.Handle(ctx, cmd)
}
