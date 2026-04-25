package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	domzoo "github.com/kevintrivedi/zoo-api/internal/domain/zoo"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/dto"
)

func TestZooFromDomain(t *testing.T) {
	t.Parallel()

	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)

	var enc1 domzoo.Enclosure
	require.NoError(t, enc1.Add(lion))
	enclosures := []domzoo.Enclosure{enc1, {}}

	got := dto.ZooFromDomain(enclosures)

	require.Len(t, got.Enclosures, 2)
	require.Len(t, got.Enclosures[0].Animals, 1)
	require.Equal(t, "lion-1", got.Enclosures[0].Animals[0].ID)
	require.Equal(t, "lion", got.Enclosures[0].Animals[0].Type)
	require.Empty(t, got.Enclosures[1].Animals)
	require.NotNil(t, got.Unplaced)
	require.Empty(t, got.Unplaced)
}

func TestZooFromDomain_EmptyInput(t *testing.T) {
	t.Parallel()

	got := dto.ZooFromDomain(nil)
	require.NotNil(t, got.Enclosures)
	require.Empty(t, got.Enclosures)
}

func TestZooFromAssignment_IncludesUnplaced(t *testing.T) {
	t.Parallel()

	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)
	snake, err := animal.New("snake-1", "snake", 10, 1, []string{"lion"})
	require.NoError(t, err)

	var enc domzoo.Enclosure
	require.NoError(t, enc.Add(lion))

	got := dto.ZooFromAssignment(appzoo.Assignment{
		Enclosures: []domzoo.Enclosure{enc},
		Unplaced: []appzoo.UnplacedAnimal{
			{Animal: snake, Reason: appzoo.ReasonIncompatible},
		},
	})

	require.Len(t, got.Unplaced, 1)
	require.Equal(t, "snake-1", got.Unplaced[0].Animal.ID)
	require.Equal(t, "snake", got.Unplaced[0].Animal.Type)
	require.Equal(t, appzoo.ReasonIncompatible, got.Unplaced[0].Reason)
}

// TestZooResponse_EmptyUnplacedRendersAsArray guards against the JSON
// encoder emitting "unplaced": null when the slice is empty — the wire
// contract is always an array (QA cases rely on callers iterating).
func TestZooResponse_EmptyUnplacedRendersAsArray(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(dto.ZooFromAssignment(appzoo.Assignment{}))
	require.NoError(t, err)
	require.Contains(t, string(body), `"unplaced":[]`)
	require.NotContains(t, string(body), `"unplaced":null`)
}
