// Package gaelog provides easy Stackdriver Logging on Google App Engine Standard second generation runtimes.
package gaelog

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/logging"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	// DefaultLogID is the default log ID of the underlying Stackdriver Logging logger. Request
	// logs are logged under the ID "request_log", so use "app_log" for consistency. To use a
	// different ID create your logger with NewWithID.
	DefaultLogID = "app_log"

	// GAEAppResourceType is the type set on the logger's MonitoredResource for App Engine apps.
	// This matches the type that App Engine itself assigns to request logs.
	GAEAppResourceType = "gae_app"

	// CloudRunResourceType is the type set on the logger's MonitoredResource for Cloud Run revisions.
	// This matches the type that Cloud Run itself assigns to request logs.
	CloudRunResourceType = "cloud_run_revision"

	traceContextHeaderName = "X-Cloud-Trace-Context"
)

var (
	metadataOnce sync.Once

	metadataProjectID    string
	metadataProjectIDErr error
)

// projectIDFromMetadataService fetches the project ID from the metadata server,
// memoizing the result for use on all but the first call.
func projectIDFromMetadataService() (string, error) {
	metadataOnce.Do(func() {
		metadataProjectID, metadataProjectIDErr = metadata.ProjectID()
	})
	return metadataProjectID, metadataProjectIDErr
}

func traceID(projectID, trace string) string {
	return fmt.Sprintf("projects/%s/traces/%s", projectID, trace)
}

type serviceInfo struct {
	projectID string
	resource  *monitoredres.MonitoredResource
}

func newServiceInfo() (serviceInfo, error) {
	// First try getting the project ID from the env var it's exposed as on App Engine.
	gaeProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if gaeProjectID != "" {
		gaeService := os.Getenv("GAE_SERVICE")
		gaeVersion := os.Getenv("GAE_VERSION")
		if gaeService == "" || gaeVersion == "" {
			return serviceInfo{}, fmt.Errorf("gaelog: $GOOGLE_CLOUD_PROJECT is set so $GAE_SERVICE and $GAE_VERSION are expected to be set, but one or both are not. Falling back to standard library log.")
		}

		return serviceInfo{
			projectID: gaeProjectID,
			resource: &monitoredres.MonitoredResource{
				Labels: map[string]string{
					"project_id": gaeProjectID,
					"module_id":  gaeService,
					"version_id": gaeVersion,
				},
				Type: GAEAppResourceType,
			},
		}, nil
	}

	// Try the metadata service for the project ID.
	crProjectID, err := projectIDFromMetadataService()
	if err != nil {
		return serviceInfo{}, err
	}

	// We got the project ID, so get and check the env vars expected to be set on Cloud Run.
	crService := os.Getenv("K_SERVICE")
	crRevision := os.Getenv("K_REVISION")
	crConfiguration := os.Getenv("K_CONFIGURATION")
	if crService == "" || crRevision == "" || crConfiguration == "" {
		return serviceInfo{}, fmt.Errorf("gaelog: the project ID was fetched from the metadata service so $K_SERVICE, $K_REVISION, and $K_CONFIGURATION are expected to be set, but one or more are not. Falling back to standard library log.")
	}

	return serviceInfo{
		projectID: crProjectID,
		resource: &monitoredres.MonitoredResource{
			Labels: map[string]string{
				"project_id":         crProjectID,
				"service_name":       crService,
				"revision_name":      crRevision,
				"configuration_name": crConfiguration,
			},
			Type: CloudRunResourceType,
		},
	}, nil
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
// If they are not present then it is initialized using environment variables present on Cloud Run:
//
//   • K_SERVICE
//   • K_REVISION
//   • K_CONFIGURATION
//   • Project ID is fetched from the metadata server, not an env var
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
	info, err := newServiceInfo()
	if err != nil {
		return &Logger{}, err
	}

	traceContext := r.Header.Get(traceContextHeaderName)
	if traceContext == "" {
		return &Logger{}, fmt.Errorf("gaelog: %s header is not set, falling back to standard library log", traceContextHeaderName)
	}

	client, err := logging.NewClient(r.Context(), fmt.Sprintf("projects/%s", info.projectID))
	if err != nil {
		return &Logger{}, err
	}

	return &Logger{
		client: client,
		logger: client.Logger(logID, options...),
		monRes: info.resource,
		trace:  traceID(info.projectID, strings.Split(traceContext, "/")[0]),
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
