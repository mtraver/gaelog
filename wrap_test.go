package gaelog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

func TestWrap(t *testing.T) {
	envVars := map[string]string{
		"GOOGLE_CLOUD_PROJECT": testProjectID,
		"GAE_SERVICE":          testServiceID,
		"GAE_VERSION":          testVersionID,
	}

	unset := setEnvVars(envVars)
	defer unset()

	expectedResource := &monitoredres.MonitoredResource{
		Labels: map[string]string{
			"module_id":  testServiceID,
			"project_id": testProjectID,
			"version_id": testVersionID,
		},
		Type: "gae_app",
	}

	handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		cv := ctx.Value(ctxKey)
		if cv == nil {
			t.Errorf("expected value for key %q, got nil", ctxKey)
			return
		}

		logger, ok := cv.(*Logger)
		if !ok {
			t.Errorf("expected var of type *Logger, got %T", cv)
			return
		}

		if diff := pretty.Compare(logger.monRes, expectedResource); diff != "" {
			t.Errorf("Unexpected result (-got +want):\n%s", diff)
			return
		}

		fmt.Fprintf(w, "ok")
	}))

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set(traceContextHeaderName, "abcdef0123456789/abcdef")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
}
