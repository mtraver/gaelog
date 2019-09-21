// Package gaelog provides easy Stackdriver Logging on Google App Engine Standard second generation runtimes.
package gaelog

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/logging"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	// DefaultLogID is the default log ID of the underlying Stackdriver Logging logger. Request
	// logs are logged under the ID "request_log", so use "app_log" for consistency. To use a
	// different ID create your logger with NewWithID.
	DefaultLogID = "app_log"

	traceContextHeaderName = "X-Cloud-Trace-Context"
)

func traceID(projectID, trace string) string {
	return fmt.Sprintf("projects/%s/traces/%s", projectID, trace)
}

type envVarError struct {
	varName string
}

func (e *envVarError) Error() string {
	return fmt.Sprintf("gaelog: %s env var is not set, falling back to standard library log", e.varName)
}

// A Logger logs messages to Stackdriver Logging (though in certain cases it may fall back to the
// standard library's "log" package; see New). Logs will be correlated with requests in Stackdriver.
type Logger struct {
	client *logging.Client
	logger *logging.Logger
	monRes *monitoredres.MonitoredResource
	trace  string
}

// NewWithID creates a new Logger. The Logger is initialized using environment variables that are
// present on App Engine:
//
//   • GOOGLE_CLOUD_PROJECT
//   • GAE_SERVICE
//   • GAE_VERSION
//
// The given log ID will be passed through to the underlying Stackdriver Logging logger.
//
// Additionally, options (of type LoggerOption, from cloud.google.com/go/logging) will be passed
// through to the underlying Stackdriver Logging logger. Note that the option CommonResource will
// have no effect because the MonitoredResource is set when each log entry is made, thus overriding
// any value set with CommonResource. This is intended: much of the value of this package is in
// setting up the MonitoredResource so that log entries correlate with requests.
//
// The Logger will be valid in all cases, even when the error is non-nil. In the case of a non-nil
// error the Logger will fall back to the standard library's "log" package. There are three cases
// in which the error will be non-nil:
//
//   1. Any of the aforementioned environment variables are not set.
//   2. The given http.Request does not have the X-Cloud-Trace-Context header.
//   3. Initialization of the underlying Stackdriver Logging client produced an error.
func NewWithID(r *http.Request, logID string, options ...logging.LoggerOption) (*Logger, error) {
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		return &Logger{}, &envVarError{"GOOGLE_CLOUD_PROJECT"}
	}

	serviceID := os.Getenv("GAE_SERVICE")
	if serviceID == "" {
		return &Logger{}, &envVarError{"GAE_SERVICE"}
	}

	versionID := os.Getenv("GAE_VERSION")
	if versionID == "" {
		return &Logger{}, &envVarError{"GAE_VERSION"}
	}

	traceContext := r.Header.Get(traceContextHeaderName)
	if traceContext == "" {
		return &Logger{}, fmt.Errorf("gaelog: %s header is not set, falling back to standard library log", traceContextHeaderName)
	}

	client, err := logging.NewClient(r.Context(), fmt.Sprintf("projects/%s", projectID))
	if err != nil {
		return &Logger{}, err
	}

	monRes := &monitoredres.MonitoredResource{
		Labels: map[string]string{
			"module_id":  serviceID,
			"project_id": projectID,
			"version_id": versionID,
		},
		Type: "gae_app",
	}

	return &Logger{
		client: client,
		logger: client.Logger(logID, options...),
		monRes: monRes,
		trace:  traceID(projectID, strings.Split(traceContext, "/")[0]),
	}, nil
}

// New is identical to NewWithID with the exception that it uses the default log ID.
func New(r *http.Request, options ...logging.LoggerOption) (*Logger, error) {
	return NewWithID(r, DefaultLogID, options...)
}

// Close closes the Logger, ensuring all logs are flushed and closing the underlying
// Stackdriver Logging client.
func (lg *Logger) Close() error {
	if lg.client != nil {
		return lg.client.Close()
	}

	return nil
}

// Logf logs with the given severity. Remaining arguments are handled in the manner of fmt.Printf.
func (lg *Logger) Logf(severity logging.Severity, format string, v ...interface{}) {
	if lg.logger == nil {
		log.Printf(format, v...)
		return
	}

	lg.logger.Log(logging.Entry{
		Timestamp: time.Now(),
		Severity:  severity,
		Payload:   fmt.Sprintf(format, v...),
		Trace:     lg.trace,
		Resource:  lg.monRes,
	})
}

// Debugf calls Logf with debug severity.
func (lg *Logger) Debugf(format string, v ...interface{}) {
	lg.Logf(logging.Debug, format, v...)
}

// Infof calls Logf with info severity.
func (lg *Logger) Infof(format string, v ...interface{}) {
	lg.Logf(logging.Info, format, v...)
}

// Noticef calls Logf with notice severity.
func (lg *Logger) Noticef(format string, v ...interface{}) {
	lg.Logf(logging.Notice, format, v...)
}

// Warningf calls Logf with warning severity.
func (lg *Logger) Warningf(format string, v ...interface{}) {
	lg.Logf(logging.Warning, format, v...)
}

// Errorf calls Logf with error severity.
func (lg *Logger) Errorf(format string, v ...interface{}) {
	lg.Logf(logging.Error, format, v...)
}

// Criticalf calls Logf with critical severity.
func (lg *Logger) Criticalf(format string, v ...interface{}) {
	lg.Logf(logging.Critical, format, v...)
}

// Alertf calls Logf with alert severity.
func (lg *Logger) Alertf(format string, v ...interface{}) {
	lg.Logf(logging.Alert, format, v...)
}

// Emergencyf calls Logf with emergency severity.
func (lg *Logger) Emergencyf(format string, v ...interface{}) {
	lg.Logf(logging.Emergency, format, v...)
}

// Log logs with the given severity. v must be either a string, or something that
// marshals via the encoding/json package to a JSON object (and not any other type
// of JSON value).
func (lg *Logger) Log(severity logging.Severity, v interface{}) {
	if lg.logger == nil {
		log.Print(v)
		return
	}

	lg.logger.Log(logging.Entry{
		Timestamp: time.Now(),
		Severity:  severity,
		Payload:   v,
		Trace:     lg.trace,
		Resource:  lg.monRes,
	})
}

// Debug calls Log with debug severity.
func (lg *Logger) Debug(v interface{}) {
	lg.Log(logging.Debug, v)
}

// Info calls Log with info severity.
func (lg *Logger) Info(v interface{}) {
	lg.Log(logging.Info, v)
}

// Notice calls Log with notice severity.
func (lg *Logger) Notice(v interface{}) {
	lg.Log(logging.Notice, v)
}

// Warning calls Log with warning severity.
func (lg *Logger) Warning(v interface{}) {
	lg.Log(logging.Warning, v)
}

// Error calls Log with error severity.
func (lg *Logger) Error(v interface{}) {
	lg.Log(logging.Error, v)
}

// Critical calls Log with critical severity.
func (lg *Logger) Critical(v interface{}) {
	lg.Log(logging.Critical, v)
}

// Alert calls Log with alert severity.
func (lg *Logger) Alert(v interface{}) {
	lg.Log(logging.Alert, v)
}

// Emergency calls Log with emergency severity.
func (lg *Logger) Emergency(v interface{}) {
	lg.Log(logging.Emergency, v)
}
