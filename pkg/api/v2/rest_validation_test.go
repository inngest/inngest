package apiv2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRESTValidation(t *testing.T) {
	ctx := context.Background()
	
	// Create HTTP handler
	handler, err := NewHTTPHandler(ctx, HTTPHandlerOptions{})
	require.NoError(t, err)

	t.Run("valid email passes validation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email": "test@example.com",
			"name":  "Test User",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v2/partner/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// The request should not fail due to validation (it will fail due to not implemented, which is expected)
		assert.Equal(t, http.StatusNotImplemented, rec.Code)
		
		// Parse the response to check it's the expected error format
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errors, ok := response["errors"].([]interface{})
		require.True(t, ok, "Response should have errors array")
		require.Greater(t, len(errors), 0, "Should have at least one error")
		
		firstError, ok := errors[0].(map[string]interface{})
		require.True(t, ok, "First error should be an object")
		
		// Should be "not implemented" error, not validation error
		assert.Equal(t, "not_implemented", firstError["code"])
	})

	t.Run("invalid email fails validation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email": "invalid-email",
			"name":  "Test User",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v2/partner/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should fail with BadRequest due to validation
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		// Parse the response to check validation error
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errors, ok := response["errors"].([]interface{})
		require.True(t, ok, "Response should have errors array")
		require.Greater(t, len(errors), 0, "Should have at least one error")
		
		firstError, ok := errors[0].(map[string]interface{})
		require.True(t, ok, "First error should be an object")
		
		// Should contain validation error message
		message := firstError["message"].(string)
		assert.Contains(t, message, "email")
		assert.Contains(t, message, "valid email address")
	})

	t.Run("empty email fails validation", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email": "",
			"name":  "Test User",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v2/partner/accounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Should fail with BadRequest due to validation
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		
		// Parse the response
		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)
		
		errors, ok := response["errors"].([]interface{})
		require.True(t, ok, "Response should have errors array")
		require.Greater(t, len(errors), 0, "Should have at least one error")
		
		firstError, ok := errors[0].(map[string]interface{})
		require.True(t, ok, "First error should be an object")
		
		// Should contain validation error message
		message := firstError["message"].(string)
		assert.Contains(t, message, "email")
		assert.Contains(t, message, "empty")
	})
}