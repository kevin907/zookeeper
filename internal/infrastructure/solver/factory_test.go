package solver_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/infrastructure/solver"
)

func TestNewSolver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		arg     string
		wantErr bool
	}{
		{"empty resolves to default", "", false},
		{"greedy resolves", "greedy", false},
		{"unknown returns error", "maxpop", true},
		{"case sensitive", "Greedy", true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := solver.NewSolver(tc.arg)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}
