package postgres

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// These tests exercise only the JSONB decoder — no real DB is needed.
// Real Postgres behaviour is covered by the integration test in Task 08.

func TestDecodeRow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload string
		wantErr bool
		check   func(t *testing.T, gotType string, popularity, maxFriends int, incompat []string)
	}{
		{
			name:    "happy path with incompatibles",
			payload: `{"type":"snake","popularity":10,"incompatible":["giraffe","tiger"],"maximumFriends":5}`,
			check: func(t *testing.T, typ string, pop, max int, inc []string) {
				require.Equal(t, "snake", typ)
				require.Equal(t, 10, pop)
				require.Equal(t, 5, max)
				require.Equal(t, []string{"giraffe", "tiger"}, inc)
			},
		},
		{
			name:    "null incompatible",
			payload: `{"type":"lion","popularity":1,"incompatible":null,"maximumFriends":3}`,
			check: func(t *testing.T, typ string, pop, max int, inc []string) {
				require.Equal(t, "lion", typ)
				require.Nil(t, inc)
			},
		},
		{
			name:    "missing incompatible key",
			payload: `{"type":"lion","popularity":1,"maximumFriends":3}`,
			check: func(t *testing.T, typ string, pop, max int, inc []string) {
				require.Equal(t, "lion", typ)
				require.Nil(t, inc)
			},
		},
		{
			name:    "malformed json",
			payload: `{"type":"snake",`,
			wantErr: true,
		},
		{
			name:    "invalid animal (zero maxFriends) rejected by constructor",
			payload: `{"type":"lion","popularity":1,"incompatible":null,"maximumFriends":0}`,
			wantErr: true,
		},
		{
			name:    "invalid animal (empty type) rejected by constructor",
			payload: `{"type":"","popularity":1,"incompatible":null,"maximumFriends":1}`,
			wantErr: true,
		},
	}

	for i, tc := range cases {
		tc, i := tc, i
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := decodeRow(int64(i+1), []byte(tc.payload))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("%d", i+1), got.ID, "decoder must thread the DB id into Animal.ID")
			tc.check(t, got.Type, got.Popularity, got.MaximumFriends, got.Incompatible)
		})
	}
}

func TestToRow_RoundTrip(t *testing.T) {
	t.Parallel()

	payload := `{"type":"tiger","popularity":8,"incompatible":["snake"],"maximumFriends":3}`
	a, err := decodeRow(42, []byte(payload))
	require.NoError(t, err)
	require.Equal(t, "42", a.ID)

	row := toRow(a)
	require.Equal(t, "tiger", row.Type)
	require.Equal(t, 8, row.Popularity)
	require.Equal(t, []string{"snake"}, row.Incompatible)
	require.Equal(t, 3, row.MaximumFriends)
}
