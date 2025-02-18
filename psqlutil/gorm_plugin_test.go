package psqlutil

import (
	"database/sql"
	"testing"

	"github.com/lib/pq"
	"github.com/purposeinplay/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGORMErrorsPlugin_handleError(t *testing.T) {
	tests := map[string]struct {
		inputErr      error
		table         string
		errorDetails  map[errors.ErrorType]map[string]errors.ErrorDetail
		expectedError *errors.Error
	}{
		"NoError": {
			inputErr:      nil,
			expectedError: nil,
		},
		"RecordNotFoundError": {
			inputErr: gorm.ErrRecordNotFound,
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeNotFound,
				Message:         "record not found",
				InternalMessage: gorm.ErrRecordNotFound.Error(),
			},
		},
		"SQLNoRowsError": {
			inputErr: sql.ErrNoRows,
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeNotFound,
				Message:         "record not found",
				InternalMessage: sql.ErrNoRows.Error(),
			},
		},
		"UniqueViolationError": {
			inputErr: &pq.Error{
				Code: "23505",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeConflict,
				Message:         "object already exists",
				InternalMessage: (&pq.Error{Code: "23505"}).Error(),
			},
		},
		"InvalidInputError": {
			inputErr: &pq.Error{
				Code: "22P02",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "invalid input",
				InternalMessage: (&pq.Error{Code: "22P02"}).Error(),
			},
		},
		"ForeignKeyViolationError": {
			inputErr: &pq.Error{
				Code: "23503",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "invalid input",
				InternalMessage: (&pq.Error{Code: "23503"}).Error(),
			},
		},
		"CheckConstraintViolationError": {
			inputErr: &pq.Error{
				Code: "23514",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "invalid input",
				InternalMessage: (&pq.Error{Code: "23514"}).Error(),
			},
		},
		"UnknownError": {
			inputErr: errors.New("unknown error"),
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInternalError,
				Message:         "unknown error",
				InternalMessage: "unknown error",
			},
		},
		"ErrorWithTableDetails": {
			inputErr: gorm.ErrRecordNotFound,
			table:    "users",
			errorDetails: map[errors.ErrorType]map[string]errors.ErrorDetail{
				errors.ErrorTypeNotFound: {
					"users": errors.ErrorDetail{
						Code:    "USER_NOT_FOUND",
						Message: "user not found",
					},
				},
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeNotFound,
				Message:         "record not found",
				InternalMessage: gorm.ErrRecordNotFound.Error(),
				ErrorDetails: errors.ErrorDetails{
					{
						Code:    "USER_NOT_FOUND",
						Message: "user not found",
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a mock DB with the test error
			db := &gorm.DB{Error: tt.inputErr}
			if tt.table != "" {
				db.Statement = &gorm.Statement{Table: tt.table}
			}

			// Create plugin instance with test error details
			plugin := &GORMErrorsPlugin{
				TableToErrorDetail: tt.errorDetails,
			}

			// Call handleError
			plugin.handleError(db)

			// Check the result
			if tt.expectedError == nil {
				assert.NoError(t, db.Error)
			} else {
				require.IsType(t, &errors.Error{}, db.Error)

				// nolint: revive
				actualErr, _ := db.Error.(*errors.Error)

				assert.Equal(t, tt.expectedError, actualErr)
			}
		})
	}
}
