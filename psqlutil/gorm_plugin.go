package psqlutil

import (
	"database/sql"

	"github.com/lib/pq"
	"github.com/purposeinplay/go-commons/errors"
	"gorm.io/gorm"
)

var _ gorm.ErrorTranslator

// GORMErrorsPlugin is a plugin for GORM that handles postgres specific errors.
type GORMErrorsPlugin struct{}

// Name returns the name of the plugin.
func (GORMErrorsPlugin) Name() string {
	return "psqlerrors"
}

// Initialize registers the plugin with the GORM db.
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

func (GORMErrorsPlugin) handleError(tx *gorm.DB) {
	err := tx.Error

	if err == nil {
		return
	}

	const (
		errCodeUniqueViolation = pq.ErrorCode("23505")
		errCodeInvalidInput    = pq.ErrorCode("22P02")
	)

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, sql.ErrNoRows):
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeNotFound,
			Message: "record not found",
		}
	case isErrorCode(err, errCodeInvalidInput):
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInvalid,
			Message: "invalid input",
		}
	case isErrorCode(err, errCodeUniqueViolation):
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInvalid,
			Message: "object already exists",
		}
	default:
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInternalError,
			Message: tx.Error.Error(),
		}
	}
}

// isErrorCode a specific postgres specific error as defined by
// https://www.postgresql.org/docs/9.5/static/errcodes-appendix.html
func isErrorCode(err error, errCode pq.ErrorCode) bool {
	var pqErr *pq.Error

	if errors.As(err, &pqErr) {
		return pqErr.Code == errCode
	}

	return false
}
