package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
	"github.com/kevin907/zookeeper/internal/interfaces/http/dto"
	"github.com/kevin907/zookeeper/internal/interfaces/http/handler"
	"github.com/kevin907/zookeeper/internal/interfaces/http/httperr"
)

type fakeAssigner struct {
	result appzoo.Assignment
	err    error
	seenN  int
}

func (f *fakeAssigner) AssignEnclosures(_ context.Context, n int) (appzoo.Assignment, error) {
	f.seenN = n
	return f.result, f.err
}

func oneEnclosure(t *testing.T) appzoo.Assignment {
	t.Helper()
	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)
	var enc domzoo.Enclosure
	require.NoError(t, enc.Add(lion))
	return appzoo.Assignment{Enclosures: []domzoo.Enclosure{enc}}
}

func sendRequest(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func mountedZoo(svc handler.ZooAssigner) http.Handler {
	r := chi.NewRouter()
	h := handler.NewZooHandler(svc)
	r.Get("/api/v1/zoos/{enclosures}", h.Get)
	return r
}

func TestZoo_Get_HappyPath(t *testing.T) {
	t.Parallel()

	svc := &fakeAssigner{result: oneEnclosure(t)}
	rec := sendRequest(t, mountedZoo(svc), "/api/v1/zoos/1")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	require.Equal(t, 1, svc.seenN)

	var body dto.ZooResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Enclosures, 1)
	require.Equal(t, "lion", body.Enclosures[0].Animals[0].Type)
	require.NotNil(t, body.Unplaced)
	require.Empty(t, body.Unplaced)
}

func TestZoo_Get_SurfacesUnplaced(t *testing.T) {
	t.Parallel()

	snake, err := animal.New("snake-1", "snake", 10, 1, []string{"lion"})
	require.NoError(t, err)
	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)
	var enc domzoo.Enclosure
	require.NoError(t, enc.Add(lion))

	svc := &fakeAssigner{result: appzoo.Assignment{
		Enclosures: []domzoo.Enclosure{enc},
		Unplaced: []appzoo.UnplacedAnimal{
			{Animal: snake, Reason: appzoo.ReasonIncompatible},
		},
	}}
	rec := sendRequest(t, mountedZoo(svc), "/api/v1/zoos/1")

	require.Equal(t, http.StatusOK, rec.Code)

	var body dto.ZooResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Unplaced, 1)
	require.Equal(t, "snake-1", body.Unplaced[0].Animal.ID)
	require.Equal(t, appzoo.ReasonIncompatible, body.Unplaced[0].Reason)
}

func TestZoo_Get_InvalidEnclosures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
	}{
		{"zero", "/api/v1/zoos/0"},
		{"negative", "/api/v1/zoos/-1"},
		{"too large", "/api/v1/zoos/9"},
		{"non-integer", "/api/v1/zoos/abc"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := &fakeAssigner{err: fmt.Errorf("%w: nope", appzoo.ErrEnclosuresOutOfRange)}
			rec := sendRequest(t, mountedZoo(svc), tc.path)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.Equal(t, "application/problem+json; charset=utf-8", rec.Header().Get("Content-Type"))

			var prob httperr.Problem
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prob))
			require.Equal(t, httperr.TypeValidation, prob.Type)
		})
	}
}

func TestZoo_Get_MapsInfeasibleTo422(t *testing.T) {
	t.Parallel()

	svc := &fakeAssigner{err: fmt.Errorf("solve: %w", appzoo.ErrInfeasibleAssignment)}
	rec := sendRequest(t, mountedZoo(svc), "/api/v1/zoos/3")

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	var prob httperr.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prob))
	require.Equal(t, httperr.TypeInfeasible, prob.Type)
}

func TestZoo_Get_MapsUnknownErrorTo500(t *testing.T) {
	t.Parallel()

	svc := &fakeAssigner{err: errors.New("db down")}
	rec := sendRequest(t, mountedZoo(svc), "/api/v1/zoos/3")

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var prob httperr.Problem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prob))
	require.Equal(t, "internal error", prob.Detail, "internal error text must not leak inner cause")
}
