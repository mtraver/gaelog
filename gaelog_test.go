package gaelog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	testProjectID = "my-project"
	testServiceID = "my-service"
	testVersionID = "my-version"

	// testProjectIDMetadataServer is a different project ID that is returned from
	// the metadata server mock so that the source of the ID may be distinguished.
	testProjectIDMetadataServer = "my-project-from-metadata-server"
)

func TestTraceID(t *testing.T) {
	got := traceID(testProjectID, "abcdef0123456789")
	expected := "projects/" + testProjectID + "/traces/abcdef0123456789"
	if got != expected {
		t.Errorf("Expected %v, got %v", expected, got)
	}
}

func TestNew(t *testing.T) {
	// Mock the metadata service.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/computeMetadata/v1/project/project-id":
			w.Write([]byte(testProjectIDMetadataServer))
		case "/computeMetadata/v1/":
			w.Write([]byte(""))
		default:
			t.Errorf("Unknown metadata server path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	// If it is set, the metadata package uses $GCE_METADATA_HOST instead of its
	// hard-coded IP of the service. The metadata package prepends the protocol
	// so strip it off here.
	defer os.Unsetenv("GCE_METADATA_HOST")
	os.Setenv("GCE_METADATA_HOST", strings.TrimPrefix(server.URL, "http://"))

	cases := []struct {
		envVars   map[string]string
		setHeader bool
		expectErr string
	}{
		{nil, false, "GOOGLE_CLOUD_PROJECT env var is not set"},
		{nil, true, "GOOGLE_CLOUD_PROJECT env var is not set"},
		{
			map[string]string{
				"GOOGLE_CLOUD_PROJECT": testProjectID,
				"GAE_SERVICE":          testServiceID,
				"GAE_VERSION":          testVersionID,
			},
			true,
			"",
		},
		{
			map[string]string{
				"GOOGLE_CLOUD_PROJECT": testProjectID,
				"GAE_SERVICE":          testServiceID,
				"GAE_VERSION":          testVersionID,
			},
			false,
			"X-Cloud-Trace-Context header is not set",
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("vars %v header %v", c.envVars != nil, c.setHeader), func(t *testing.T) {
			if c.envVars != nil {
				defer func() {
					for k, _ := range c.envVars {
						os.Unsetenv(k)
					}
				}()

				for k, v := range c.envVars {
					os.Setenv(k, v)
				}
			}

			r := httptest.NewRequest("GET", "https://example.com", nil)
			if c.setHeader {
				r.Header.Set(traceContextHeaderName, "abcdef0123456789/abcdef")
			}

			lg, err := New(r)
			if c.expectErr == "" && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			} else if c.expectErr != "" && err == nil {
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
			if c.expectErr == "" && err == nil {
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

			if c.expectErr != "" && err != nil {
				if !strings.Contains(err.Error(), c.expectErr) {
					t.Errorf("Expected error to contain %q, got %q", c.expectErr, err)
				}
			}
		})
	}
}
