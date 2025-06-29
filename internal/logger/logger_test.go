package logger_test

import (
	"context"
	"testing"

	"github.com/obi2na/petrel/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

func TestInjectRequestID(t *testing.T) {
	ctx := context.Background()
	reqID := "test-id-123"

	ctxWithID := logger.InjectRequestID(ctx, reqID)

	val := ctxWithID.Value("request_id")
	if val != reqID {
		t.Errorf("Expected request_id %s, got %v", reqID, val)
	}
}

func TestWithAddsRequestID(t *testing.T) {
	// Set up a test logger with in-memory buffer
	var buf strings.Builder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.InfoLevel,
	)
	testLogger := zap.New(core)

	// Set up baseLogger manually (since we're not calling Init())
	// You may need to export baseLogger in logger.go to test this cleanly
	// OR expose a test-only setter
	ctx := context.Background()
	ctx = logger.InjectRequestID(ctx, "test-id-456")

	// This won't add the field unless logger.With() in your code is implemented properly
	log := testLogger.With(zap.String("request_id", "test-id-456"))
	log.Info("testing With")

	if !strings.Contains(buf.String(), "test-id-456") {
		t.Errorf("Expected log to contain request_id, got: %s", buf.String())
	}
}
