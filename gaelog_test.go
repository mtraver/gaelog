package gaelog

import (
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	testProjectID = "my-project"
	testServiceID = "my-service"
	testVersionID = "my-version"
)

// setVars ensures that all of the required environment variables are set.
func setVars(t *testing.T) {
	t.Helper()
	os.Setenv("GOOGLE_CLOUD_PROJECT", testProjectID)
	os.Setenv("GAE_SERVICE", testServiceID)
	os.Setenv("GAE_VERSION", testVersionID)
}

// unsetVars ensures that none of the required environment variables are set.
func unsetVars(t *testing.T) {
	t.Helper()
	os.Unsetenv("GOOGLE_CLOUD_PROJECT")
	os.Unsetenv("GAE_SERVICE")
	os.Unsetenv("GAE_VERSION")
}

func TestTraceID(t *testing.T) {
	got := traceID(testProjectID, "abcdef0123456789")
	expected := "projects/" + testProjectID + "/traces/abcdef0123456789"
	if got != expected {
		t.Errorf("Expected %v, got %v", expected, got)
	}
}

func TestNew(t *testing.T) {
	cases := []struct {
		setVars   bool
		setHeader bool
		expectErr bool
	}{
		{false, false, true},
		{false, true, true},
		{true, true, false},
		{true, false, true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("vars %v header %v", c.setVars, c.setHeader), func(t *testing.T) {
			unsetVars(t)

			r := httptest.NewRequest("GET", "https://example.com", nil)
			if c.setHeader {
				r.Header.Set(traceContextHeaderName, "abcdef0123456789/abcdef")
			}

			if c.setVars {
				setVars(t)
			}

			lg, err := New(r)
			if !c.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			} else if c.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
				return
			}

			expectedMonRes := &monitoredres.MonitoredResource{
				Labels: map[string]string{
					"module_id":  testServiceID,
					"project_id": testProjectID,
					"version_id": testVersionID,
				},
				Type: "gae_app",
			}
			if !c.expectErr && err == nil {
				if diff := pretty.Compare(lg.monRes, expectedMonRes); diff != "" {
					t.Errorf("Unexpected result (-got +want):\n%s", diff)
				}

				if lg.client == nil {
					t.Errorf("Client is nil")
				}

				if lg.logger == nil {
					t.Errorf("Logger is nil")
				}
			}
		})
	}
}
