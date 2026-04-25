package httperr_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/httperr"
)

func TestMap(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		err        error
		wantStatus int
		wantType   string
	}{
		{"enclosures out of range", fmt.Errorf("%w: got 0", appzoo.ErrEnclosuresOutOfRange),
			http.StatusBadRequest, httperr.TypeValidation},
		{"invalid enclosures input (non-integer)", fmt.Errorf("%w: %q is not an integer", appzoo.ErrInvalidEnclosuresInput, "abc"),
			http.StatusBadRequest, httperr.TypeValidation},
		{"infeasible assignment", fmt.Errorf("solve: %w", appzoo.ErrInfeasibleAssignment),
			http.StatusUnprocessableEntity, httperr.TypeInfeasible},
		{"dependency unavailable", fmt.Errorf("readyz: %w", appzoo.ErrDependencyUnavailable),
			http.StatusServiceUnavailable, httperr.TypeUnavailable},
		{"unknown error falls through to internal", errors.New("boom"),
			http.StatusInternalServerError, httperr.TypeInternal},
		{"panic wrap falls through to internal", fmt.Errorf("panic: %v", "kaboom"),
			http.StatusInternalServerError, httperr.TypeInternal},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := httperr.Map(tc.err)
			require.Equal(t, tc.wantStatus, p.Status)
			require.Equal(t, tc.wantType, p.Type)
		})
	}
}

func TestWrite_SetsContentTypeAndEncodes(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httperr.Write(rec, httperr.Map(fmt.Errorf("%w: got 9", appzoo.ErrEnclosuresOutOfRange)))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "application/problem+json; charset=utf-8", rec.Header().Get("Content-Type"))

	var got httperr.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, httperr.TypeValidation, got.Type)
}

func TestMap_HidesInternalErrorDetail(t *testing.T) {
	t.Parallel()

	p := httperr.Map(errors.New("connection string leaked here"))
	require.NotContains(t, p.Detail, "connection string")
	require.Equal(t, "internal error", p.Detail)
}
