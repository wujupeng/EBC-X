package handler

import (
	"context"
	"fmt"

	"github.com/wujupeng/ebcx/internal/person"
)

type PersonCommandBus struct {
	createHandler *CreatePersonHandler
	updateHandler *UpdatePersonHandler
	assignHandler *AssignRoleHandler
}

func NewPersonCommandBus(
	createHandler *CreatePersonHandler,
	updateHandler *UpdatePersonHandler,
	assignHandler *AssignRoleHandler,
) *PersonCommandBus {
	return &PersonCommandBus{
		createHandler: createHandler,
		updateHandler: updateHandler,
		assignHandler: assignHandler,
	}
}

func (b *PersonCommandBus) DispatchCreate(ctx context.Context, cmd person.CreatePersonCommand) (*person.PersonAggregate, error) {
	return b.createHandler.Handle(ctx, cmd)
}

func (b *PersonCommandBus) DispatchUpdate(ctx context.Context, cmd person.UpdatePersonCommand) (*person.PersonAggregate, error) {
	return b.updateHandler.Handle(ctx, cmd)
}

func (b *PersonCommandBus) DispatchAssignRole(ctx context.Context, cmd person.AssignRoleCommand) (*person.PersonAggregate, error) {
	return b.assignHandler.Handle(ctx, cmd)
}

func (b *PersonCommandBus) Dispatch(ctx context.Context, cmd interface{}) (*person.PersonAggregate, error) {
	switch c := cmd.(type) {
	case person.CreatePersonCommand:
		return b.createHandler.Handle(ctx, c)
	case person.UpdatePersonCommand:
		return b.updateHandler.Handle(ctx, c)
	case person.AssignRoleCommand:
		return b.assignHandler.Handle(ctx, c)
	default:
		return nil, fmt.Errorf("%w: %T", person.ErrPersonUnknownCommand, cmd)
	}
}
