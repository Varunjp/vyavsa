package repository

import "context"

// Transactor defines the contract for executing multi-step operations within an atomic transaction
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
