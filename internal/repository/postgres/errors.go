package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MapDBError converts PostgreSQL and driver errors into structured application AppErrors.
// It ensures raw database error details, SQL queries, and driver messages are never exposed to API clients.
func MapDBError(err error, defaultOp string) error {
	if err == nil {
		return nil
	}

	// If already an AppError, preserve it
	var appErr *appErrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Record not found
	if errors.Is(err, pgx.ErrNoRows) {
		return appErrors.NewNotFound(fmt.Sprintf("%s: record not found", defaultOp), err)
	}

	// Context cancellations and deadlines
	if errors.Is(err, context.Canceled) {
		return appErrors.New("REQUEST_CANCELED", "database operation was canceled", 499, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return appErrors.New(appErrors.CodeTimeout, "database operation timed out", http.StatusGatewayTimeout, err)
	}

	// PostgreSQL SQLSTATE error codes (pgconn.PgError)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			msg := "resource with this information already exists"
			if pgErr.ConstraintName != "" {
				msg = fmt.Sprintf("conflict on unique constraint: %s", pgErr.ConstraintName)
			}
			return appErrors.NewConflict(msg, err)
		case "23503": // foreign_key_violation
			return appErrors.NewBadRequest("referenced resource does not exist or is currently in use", err)
		case "23502": // not_null_violation
			return appErrors.NewBadRequest("required database field is missing", err)
		case "23514": // check_violation
			return appErrors.NewBadRequest("data validation constraint violated", err)
		case "40001": // serialization_failure
			return appErrors.NewConflict("concurrent transaction conflict, please retry operation", err)
		case "40P01": // deadlock_detected
			return appErrors.NewConflict("database deadlock detected, please retry operation", err)
		case "57014": // query_canceled (e.g. statement_timeout)
			return appErrors.New(appErrors.CodeTimeout, "database query execution timed out", http.StatusGatewayTimeout, err)
		}
	}

	// Unexpected database internal error - wraps original error for server logs but safe generic message for client
	return appErrors.NewDatabase(fmt.Errorf("%s: %w", defaultOp, err))
}
