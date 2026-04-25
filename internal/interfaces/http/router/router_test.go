package router_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	domzoo "github.com/kevintrivedi/zoo-api/internal/domain/zoo"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/handler"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/router"
)

type fakeAssigner struct{}

func (fakeAssigner) AssignEnclosures(_ context.Context, _ int) (appzoo.Assignment, error) {
	return appzoo.Assignment{Enclosures: []domzoo.Enclosure{}}, nil
}

type fakePinger struct{}

func (fakePinger) Ping(_ context.Context) error { return nil }

func newRouter(t *testing.T) http.Handler {
	t.Helper()
	return router.New(
		handler.NewZooHandler(fakeAssigner{}),
		fakePinger{},
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		1*time.Second,
		250*time.Millisecond,
	)
}

func TestRouter_RegistersKnownRoutes(t *testing.T) {
	t.Parallel()
	r := newRouter(t)

	cases := []struct {
		path       string
		wantStatus int
	}{
		{"/healthz", http.StatusOK},
		{"/readyz", http.StatusOK},
		{"/api/v1/zoos/3", http.StatusOK},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.Equal(t, tc.wantStatus, rec.Code)
		})
	}
}

func TestRouter_UnknownRoutes404(t *testing.T) {
	t.Parallel()
	r := newRouter(t)

	cases := []string{"/api/v2/zoos/3", "/nothing"}
	for _, p := range cases {
		p := p
		t.Run(p, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
			require.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}
