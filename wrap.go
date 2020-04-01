package gaelog

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/logging"
)

type ctxKeyType string

var ctxKey = ctxKeyType("gaelog-logger")

// WrapWithID wraps a handler such that the request's context may be used to call the package-level logging functions.
// See NewWithID for details on this function's arguments and how the logger is created.
func WrapWithID(h http.Handler, logID string, options ...logging.LoggerOption) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger, _ := NewWithID(r, logID, options...)
		defer logger.Close()

		ctx := context.WithValue(r.Context(), ctxKey, logger)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Wrap is identical to WrapWithID with the exception that it uses the default log ID.
func Wrap(h http.Handler, options ...logging.LoggerOption) http.Handler {
	return WrapWithID(h, DefaultLogID, options...)
}

// Logf logs with the given severity. Remaining arguments are handled in the manner of fmt.Printf.
// This should be called from a handler that has been wrapped with Wrap or WrapWithID. If it is
// called from a handler that has not been wrapped then messages are simply logged using the standard
// library's log package.
func Logf(ctx context.Context, severity logging.Severity, format string, v ...interface{}) {
	cv := ctx.Value(ctxKey)
	if cv == nil {
		// No logger in the context, so the handler wasn't wrapped.
		log.Printf(format, v...)
		return
	}

	logger := cv.(*Logger)
	logger.Logf(severity, format, v...)
}

// Debugf calls Logf with debug severity.
func Debugf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Debug, format, v...)
}

// Infof calls Logf with info severity.
func Infof(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Info, format, v...)
}

// Noticef calls Logf with notice severity.
func Noticef(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Notice, format, v...)
}

// Warningf calls Logf with warning severity.
func Warningf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Warning, format, v...)
}

// Errorf calls Logf with error severity.
func Errorf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Error, format, v...)
}

// Criticalf calls Logf with critical severity.
func Criticalf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Critical, format, v...)
}

// Alertf calls Logf with alert severity.
func Alertf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Alert, format, v...)
}

// Emergencyf calls Logf with emergency severity.
func Emergencyf(ctx context.Context, format string, v ...interface{}) {
	Logf(ctx, logging.Emergency, format, v...)
}

// Log logs with the given severity. v must be either a string, or something that
// marshals via the encoding/json package to a JSON object (and not any other type
// of JSON value). This should be called from a handler that has been wrapped with
// Wrap or WrapWithID. If it is called from a handler that has not been wrapped
// then messages are simply logged using the standard library's log package.
func Log(ctx context.Context, severity logging.Severity, v interface{}) {
	cv := ctx.Value(ctxKey)
	if cv == nil {
		// No logger in the context, so the handler wasn't wrapped.
		log.Print(v)
		return
	}

	logger := cv.(*Logger)
	logger.Log(severity, v)
}

// Debug calls Log with debug severity.
func Debug(ctx context.Context, v interface{}) {
	Log(ctx, logging.Debug, v)
}

// Info calls Log with info severity.
func Info(ctx context.Context, v interface{}) {
	Log(ctx, logging.Info, v)
}

// Notice calls Log with notice severity.
func Notice(ctx context.Context, v interface{}) {
	Log(ctx, logging.Notice, v)
}

// Warning calls Log with warning severity.
func Warning(ctx context.Context, v interface{}) {
	Log(ctx, logging.Warning, v)
}

// Error calls Log with error severity.
func Error(ctx context.Context, v interface{}) {
	Log(ctx, logging.Error, v)
}

// Critical calls Log with critical severity.
func Critical(ctx context.Context, v interface{}) {
	Log(ctx, logging.Critical, v)
}

// Alert calls Log with alert severity.
func Alert(ctx context.Context, v interface{}) {
	Log(ctx, logging.Alert, v)
}

// Emergency calls Log with emergency severity.
func Emergency(ctx context.Context, v interface{}) {
	Log(ctx, logging.Emergency, v)
}
