package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStructuredLogger(t *testing.T) {
	t.Run("Contextual attributes propagation in JSON mode", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := New(Config{
			Environment: "production",
			ServiceName: "test-service",
			Level:       "info",
			Format:      "json",
			Output:      buf,
		})

		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-12345")
		ctx = WithTenantID(ctx, "tenant-67890")
		ctx = WithUserID(ctx, "user-abc")

		log.InfoContext(ctx, "test event occurred", "extra_key", "extra_value")

		var logMap map[string]any
		err := json.Unmarshal(buf.Bytes(), &logMap)
		assert.NoError(t, err)

		assert.Equal(t, "test-service", logMap["service"])
		assert.Equal(t, "production", logMap["environment"])
		assert.Equal(t, "req-12345", logMap["request_id"])
		assert.Equal(t, "tenant-67890", logMap["tenant_id"])
		assert.Equal(t, "user-abc", logMap["user_id"])
		assert.Equal(t, "extra_value", logMap["extra_key"])
		assert.Equal(t, "test event occurred", logMap["msg"])
	})
}
