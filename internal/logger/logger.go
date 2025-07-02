package logger

import (
	"context"
	"github.com/obi2na/petrel/config"
	"go.uber.org/zap"
	"log"
)

var baseLogger *zap.Logger

const requestIDKey string = "request_id"

func Init() {
	var err error
	if config.C.Env != "prod" {
		cfg := zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
		baseLogger, err = cfg.Build()
	} else {
		baseLogger, err = zap.NewProduction()
	}
	if err != nil {
		panic("cannot initialize Zap logger: " + err.Error())
	}
}

// returns a logger with request_id context if available
func With(ctx context.Context) *zap.Logger {
	reqId, ok := ctx.Value(requestIDKey).(string)
	if !ok || reqId == "" {
		log.Println("ReqId logger unable to be created, using base logger")
		return baseLogger
	}
	return baseLogger.With(zap.String(requestIDKey, reqId))
}

// InjectRequestID stores the ID in context after wrapping it
func InjectRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
