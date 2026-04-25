package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/handler"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/httperr"
)

type fakePinger struct{ err error }

func (f *fakePinger) Ping(_ context.Context) error { return f.err }

func TestLiveness_Returns200(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	handler.Liveness(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"ok"`)
}

func TestReadiness_Returns200WhenPingerHealthy(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	handler.Readiness(&fakePinger{}, 250*time.Millisecond)(rec,
		httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestReadiness_Returns503WhenPingerFails(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	handler.Readiness(&fakePinger{err: errors.New("down")}, 250*time.Millisecond)(rec,
		httptest.NewRequest(http.MethodGet, "/readyz", nil))

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "application/problem+json; charset=utf-8", rec.Header().Get("Content-Type"))

	var body httperr.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, httperr.TypeUnavailable, body.Type, "readiness failure must route through httperr.Map")
}
