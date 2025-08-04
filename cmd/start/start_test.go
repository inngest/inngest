package start

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// validatePostgreSQLConnectionPool extracts the validation logic from the action function
// to make it testable in isolation
func validatePostgreSQLConnectionPool(maxIdleConns, maxOpenConns int) error {
	if maxOpenConns <= 1 {
		return &PostgreSQLValidationError{
			message: fmt.Sprintf("postgres-max-open-conns (%d) must be greater than 1", maxOpenConns),
		}
	}
	if maxIdleConns > maxOpenConns {
		return &PostgreSQLValidationError{
			message: fmt.Sprintf("postgres-max-idle-conns (%d) cannot be greater than postgres-max-open-conns (%d)",
				maxIdleConns, maxOpenConns),
		}
	}
	return nil
}

// PostgreSQLValidationError represents a PostgreSQL connection pool validation error
type PostgreSQLValidationError struct {
	message string
}

func (e *PostgreSQLValidationError) Error() string {
	return e.message
}

func TestValidatePostgreSQLConnectionPool(t *testing.T) {
	tests := []struct {
		name                  string
		postgresMaxIdleConns  int
		postgresMaxOpenConns  int
		expectError           bool
		expectedErrorContains string
	}{
		{
			name:                 "valid connection pool settings",
			postgresMaxIdleConns: 10,
			postgresMaxOpenConns: 100,
			expectError:          false,
		},
		{
			name:                  "max open connections too low (0)",
			postgresMaxIdleConns:  10,
			postgresMaxOpenConns:  0,
			expectError:           true,
			expectedErrorContains: "postgres-max-open-conns (0) must be greater than 1",
		},
		{
			name:                  "max open connections too low (1)",
			postgresMaxIdleConns:  10,
			postgresMaxOpenConns:  1,
			expectError:           true,
			expectedErrorContains: "postgres-max-open-conns (1) must be greater than 1",
		},
		{
			name:                 "max open connections at boundary (2)",
			postgresMaxIdleConns: 1,
			postgresMaxOpenConns: 2,
			expectError:          false,
		},
		{
			name:                  "max idle connections greater than max open connections",
			postgresMaxIdleConns:  50,
			postgresMaxOpenConns:  25,
			expectError:           true,
			expectedErrorContains: "postgres-max-idle-conns (50) cannot be greater than postgres-max-open-conns (25)",
		},
		{
			name:                 "max idle connections equal to max open connections",
			postgresMaxIdleConns: 25,
			postgresMaxOpenConns: 25,
			expectError:          false,
		},
		{
			name:                 "zero max idle connections with valid max open connections",
			postgresMaxIdleConns: 0,
			postgresMaxOpenConns: 10,
			expectError:          false,
		},
		{
			name:                 "default values should be valid",
			postgresMaxIdleConns: 10,
			postgresMaxOpenConns: 100,
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePostgreSQLConnectionPool(tt.postgresMaxIdleConns, tt.postgresMaxOpenConns)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrorContains != "" {
					assert.Contains(t, err.Error(), tt.expectedErrorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}