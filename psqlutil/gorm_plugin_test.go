package psqlutil

import (
	"database/sql"
	"testing"

	"github.com/lib/pq"
	"github.com/purposeinplay/go-commons/errors"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGORMErrorsPlugin_handleError(t *testing.T) {
	tests := []struct {
		name          string
		inputErr      error
		table         string
		errorDetails  map[errors.ErrorType]map[string]errors.ErrorDetail
		expectedError *errors.Error
	}{
		{
			name:          "no error",
			inputErr:      nil,
			expectedError: nil,
		},
		{
			name:     "record not found error",
			inputErr: gorm.ErrRecordNotFound,
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeNotFound,
				Message:         "record not found",
				InternalMessage: gorm.ErrRecordNotFound.Error(),
			},
		},
		{
			name:     "sql no rows error",
			inputErr: sql.ErrNoRows,
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeNotFound,
				Message:         "record not found",
				InternalMessage: sql.ErrNoRows.Error(),
			},
		},
		{
			name: "unique violation error",
			inputErr: &pq.Error{
				Code: "23505",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "object already exists",
				InternalMessage: (&pq.Error{Code: "23505"}).Error(),
			},
		},
		{
			name: "invalid input error",
			inputErr: &pq.Error{
				Code: "22P02",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "invalid input",
				InternalMessage: (&pq.Error{Code: "22P02"}).Error(),
			},
		},
		{
			name: "foreign key violation error",
			inputErr: &pq.Error{
				Code: "23503",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "foreign key violation",
				InternalMessage: (&pq.Error{Code: "23503"}).Error(),
			},
		},
		{
			name: "check constraint violation error",
			inputErr: &pq.Error{
				Code: "23514",
			},
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInvalid,
				Message:         "check constraint violation",
				InternalMessage: (&pq.Error{Code: "23514"}).Error(),
			},
		},
		{
			name:     "unknown error",
			inputErr: errors.New("unknown error"),
			expectedError: &errors.Error{
				Type:            errors.ErrorTypeInternalError,
				Message:         "unknown error",
				InternalMessage: "unknown error",
			},
		},
		{
			name:     "error with table details",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				assert.Nil(t, db.Error)
			} else {
				actualErr, ok := db.Error.(*errors.Error)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedError.Type, actualErr.Type)
				assert.Equal(t, tt.expectedError.Message, actualErr.Message)
				assert.Equal(t, tt.expectedError.InternalMessage, actualErr.InternalMessage)
				assert.Equal(t, tt.expectedError.ErrorDetails, actualErr.ErrorDetails)
			}
		})
	}
}
