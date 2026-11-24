package main

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// serveUnix serves handler on a Unix socket in a temporary directory and returns its path.
func serveUnix(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "unix.socket")

	listener, err := net.Listen("unix", path)
	require.NoError(t, err)

	server := &http.Server{Handler: handler}

	go func() { _ = server.Serve(listener) }()

	t.Cleanup(func() { _ = server.Close() })

	return path
}

func TestFetchRawMetrics(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
		status      int
		body        string
		expected    string
	}{
		{
			name:        "Text format is returned as is",
			contentType: "text/plain; version=0.0.4; charset=utf-8",
			status:      http.StatusOK,
			body:        "# TYPE incusos_info gauge\nincusos_info{version=\"1\"} 1\n",
			expected:    "# TYPE incusos_info gauge\nincusos_info{version=\"1\"} 1\n",
		},
		{
			name:        "Missing trailing newline is added",
			contentType: "text/plain; version=0.0.4",
			status:      http.StatusOK,
			body:        "incusos_update_needs_reboot 0",
			expected:    "incusos_update_needs_reboot 0\n",
		},
		{
			name:        "Missing endpoint on older IncusOS is ignored",
			contentType: "application/json",
			status:      http.StatusNotFound,
			body:        `{"type":"error","error_code":404}`,
		},
		{
			name:        "OpenMetrics format is ignored",
			contentType: "application/openmetrics-text; version=1.0.0",
			status:      http.StatusOK,
			body:        "incusos_info 1\n# EOF\n",
		},
		{
			name:        "Empty response is ignored",
			contentType: "text/plain; version=0.0.4",
			status:      http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotAccept string

			path := serveUnix(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAccept = r.Header.Get("Accept")

				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})

			data := fetchRawMetrics(context.Background(), "unix", path, "/1.0/metrics")

			require.Equal(t, "/1.0/metrics", gotPath)
			require.Equal(t, "text/plain; version=0.0.4", gotAccept)

			if tc.expected == "" {
				require.Nil(t, data)
			} else {
				require.Equal(t, tc.expected, string(data))
			}
		})
	}
}

func TestFetchRawMetricsUnreachable(t *testing.T) {
	data := fetchRawMetrics(context.Background(), "unix", filepath.Join(t.TempDir(), "missing.socket"), "/1.0/metrics")
	require.Nil(t, data)
}
