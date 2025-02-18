package psqlutil

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/purposeinplay/go-commons/errors"
	"gorm.io/gorm"
)

// Ensure GORMErrorsPlugin implements gorm.Plugin interface.
var _ gorm.Plugin = (*GORMErrorsPlugin)(nil)

// GORMErrorsPlugin is a plugin for GORM that handles PostgreSQL-specific errors.
// It translates postgres-specific errors into application-specific error types
// and provides detailed error information based on the table where the error occurred.
type GORMErrorsPlugin struct {
	// TableToErrorDetail maps error types to table-specific error details.
	// The first key is the error type (e.g., ErrorTypeNotFound),
	// the second key is the table name, and the value is the specific error detail
	// to be returned for that combination.
	TableToErrorDetail map[errors.ErrorType]map[string]errors.ErrorDetail
}

// PostgreSQL error codes as defined in
// https://www.postgresql.org/docs/current/errcodes-appendix.html
// nolint: revive
const (
	// errCodeUniqueViolation is returned when a unique constraint is violated (error code 23505).
	errCodeUniqueViolation = pq.ErrorCode("23505")
	// errCodeInvalidInput is returned when the input syntax is invalid (error code 22P02).
	errCodeInvalidInput = pq.ErrorCode("22P02")
	// errCodeForeignKeyViolation is returned when a foreign key constraint is violated (error code 23503).
	errCodeForeignKeyViolation = pq.ErrorCode("23503")
	// errCodeCheckViolation is returned when a check constraint is violated (error code 23514).
	errCodeCheckViolation = pq.ErrorCode("23514")
)

// Name returns the name of the plugin.
// This is used by GORM for plugin identification.
func (GORMErrorsPlugin) Name() string {
	return "psqlerrors"
}

// Initialize registers the plugin's error handling callbacks with the GORM db.
// It adds error handling after all major database operations
// (create, query, delete, update, row, raw).
// Returns an error if any callback registration fails.
func (p GORMErrorsPlugin) Initialize(db *gorm.DB) (err error) {
	cb := db.Callback()

	return errors.Join(
		cb.Create().After("gorm:create").Register("errors:after:create", p.handleError),
		cb.Query().After("gorm:query").Register("errors:after:select", p.handleError),
		cb.Delete().After("gorm:delete").Register("errors:after:delete", p.handleError),
		cb.Update().After("gorm:update").Register("errors:after:update", p.handleError),
		cb.Row().After("gorm:row").Register("errors:after:row", p.handleError),
		cb.Raw().After("gorm:raw").Register("errors:after:raw", p.handleError),
	)
}

// handleError processes database errors and converts them into application-specific error types.
// It handles common PostgreSQL errors such as:
// - Record not found
// - Invalid input syntax
// - Unique constraint violations
// - Foreign key violations
// - Check constraint violations
// The error is enriched with table-specific error details if configured in TableToErrorDetail.
func (p GORMErrorsPlugin) handleError(tx *gorm.DB) {
	err := tx.Error

	if err == nil {
		return
	}

	txErr := &errors.Error{
		InternalMessage: tx.Error.Error(),
	}

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, sql.ErrNoRows):
		txErr.Type = errors.ErrorTypeNotFound
		txErr.Message = "record not found"

	case isErrorCode(err, errCodeInvalidInput),
		isErrorCode(err, errCodeForeignKeyViolation),
		isErrorCode(err, errCodeCheckViolation):
		txErr.Type = errors.ErrorTypeInvalid
		txErr.Message = "invalid input"

	case isErrorCode(err, errCodeUniqueViolation):
		txErr.Type = errors.ErrorTypeConflict
		txErr.Message = "object already exists"

	default:
		txErr.Type = errors.ErrorTypeInternalError
		txErr.Message = tx.Error.Error()
	}

	if ted := p.TableToErrorDetail[txErr.Type]; ted != nil {
		if ed := ted[tx.Statement.Table]; ed != (errors.ErrorDetail{}) {
			txErr.ErrorDetails = errors.ErrorDetails{ed}
		}
	}

	tx.Error = txErr
}

// isErrorCode checks if the given error matches a specific PostgreSQL error code.
// It safely unwraps the error to check if it's a *pq.Error and compares its code
// with the provided error code.
func isErrorCode(err error, errCode pq.ErrorCode) bool {
	var pqErr *pq.Error

	if errors.As(err, &pqErr) {
		return pqErr.Code == errCode
	}

	return false
}
