package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSProxyErrorHandler(t *testing.T) {
	// A failed request is reported to the client.
	rec := httptest.NewRecorder()
	osProxyErrorHandler(rec, httptest.NewRequest(http.MethodGet, "/os/1.0", nil), errors.New("dial failed"))
	require.Equal(t, http.StatusBadGateway, rec.Code)

	// Nothing is written once the client went away.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rec = httptest.NewRecorder()
	osProxyErrorHandler(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, "/os/1.0", nil), context.Canceled)
	require.False(t, rec.Flushed)
	require.Empty(t, rec.Body.String())
	require.Empty(t, rec.Header())
	require.Equal(t, http.StatusOK, rec.Code)
}
