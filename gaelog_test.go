package gaelog

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"google.golang.org/genproto/googleapis/api/monitoredres"
)

const (
	testProjectID         = "my-project"
	testServiceID         = "my-service"
	testVersionID         = "my-version"
	testConfigurationName = "my-config"

	// testProjectIDMetadataServer is a different project ID that is returned from
	// the metadata server mock so that the source of the ID may be distinguished.
	testProjectIDMetadataServer = "my-project-from-metadata-server"
)

func setEnvVars(vars map[string]string) func() {
	for k, v := range vars {
		os.Setenv(k, v)
	}

	return func() {
		for k, _ := range vars {
			os.Unsetenv(k)
		}
	}
}

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
		name           string
		envVars        map[string]string
		setHeader      bool
		expectResource *monitoredres.MonitoredResource
		expectErr      string
	}{
		{"no_env_vars_without_header", nil, false, nil, "GAE env vars were not set so Cloud Run vars"},
		{"no_env_vars_with_header", nil, true, nil, "GAE env vars were not set so Cloud Run vars"},
		{
			"gae_env_vars_with_header",
			map[string]string{
				"GOOGLE_CLOUD_PROJECT": testProjectID,
				"GAE_SERVICE":          testServiceID,
				"GAE_VERSION":          testVersionID,
			},
			true,
			&monitoredres.MonitoredResource{
				Labels: map[string]string{
					"module_id":  testServiceID,
					"project_id": testProjectID,
					"version_id": testVersionID,
				},
				Type: "gae_app",
			},
			"",
		},
		{
			"incomplete_gae_env_vars_with_header",
			map[string]string{
				"GOOGLE_CLOUD_PROJECT": testProjectID,
				"GAE_SERVICE":          testServiceID,
			},
			true,
			nil,
			"$GAE_SERVICE and $GAE_VERSION are expected to be set",
		},
		{
			"gae_env_vars_without_header",
			map[string]string{
				"GOOGLE_CLOUD_PROJECT": testProjectID,
				"GAE_SERVICE":          testServiceID,
				"GAE_VERSION":          testVersionID,
			},
			false,
			nil,
			"X-Cloud-Trace-Context header is not set",
		},

		{
			"cloud_run_env_vars_with_header",
			map[string]string{
				"K_SERVICE":       testServiceID,
				"K_REVISION":      testVersionID,
				"K_CONFIGURATION": testConfigurationName,
			},
			true,
			&monitoredres.MonitoredResource{
				Labels: map[string]string{
					"configuration_name": testConfigurationName,
					"project_id":         testProjectIDMetadataServer,
					"revision_name":      testVersionID,
					"service_name":       testServiceID,
				},
				Type: "cloud_run_revision",
			},
			"",
		},
		{
			"incomplete_cloud_run_env_vars_with_header",
			map[string]string{
				"K_SERVICE": testServiceID,
			},
			true,
			nil,
			"$K_SERVICE, $K_REVISION, and $K_CONFIGURATION are expected to be set",
		},
		{
			"cloud_run_env_vars_without_header",
			map[string]string{
				"K_SERVICE":       testServiceID,
				"K_REVISION":      testVersionID,
				"K_CONFIGURATION": testConfigurationName,
			},
			false,
			nil,
			"X-Cloud-Trace-Context header is not set",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.envVars != nil {
				unset := setEnvVars(c.envVars)
				defer unset()
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

			if c.expectErr == "" && err == nil {
				if diff := pretty.Compare(lg.monRes, c.expectResource); diff != "" {
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
