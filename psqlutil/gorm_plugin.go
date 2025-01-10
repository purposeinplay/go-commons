package psqlutil

import (
	"gorm.io/gorm"
	"github.com/lib/pq"
	"database/sql"
	"github.com/purposeinplay/go-commons/errors"
)

type GORMErrorsPlugin struct{}

func (p GORMErrorsPlugin) Name() string {
	return "psqlerrors"
}

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

func (p GORMErrorsPlugin) handleError(tx *gorm.DB) {
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
			Details: "record not found",
		}
	case isErrorCode(err, errCodeInvalidInput):
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInvalid,
			Details: "invalid input",
		}
	case isErrorCode(err, errCodeUniqueViolation):
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInvalid,
			Details: "object already exists",
		}
	default:
		tx.Error = &errors.Error{
			Type:    errors.ErrorTypeInternalError,
			Details: tx.Error.Error(),
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
