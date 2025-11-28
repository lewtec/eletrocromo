package eletrocromo

import "context"

type Task interface {
	Run(context.Context) error
}

type FunctionTask func(context.Context) error

func (f FunctionTask) Run(ctx context.Context) error {
	return f(ctx)
}
