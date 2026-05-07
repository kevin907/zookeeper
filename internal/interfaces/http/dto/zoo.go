// Package dto holds HTTP request and response shapes and their converters.
// Domain types never leak to the wire.
package dto

import (
	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
)

// ZooResponse is the body returned from GET /api/v1/zoos/{enclosures}.
// Unplaced surfaces animals the solver could not place along with a reason
// code — QA cases 1-7 ("reasonable expected behaviour").
type ZooResponse struct {
	Enclosures []EnclosureDTO `json:"enclosures"`
	Unplaced   []UnplacedDTO  `json:"unplaced"`
}

// EnclosureDTO is one entry in ZooResponse.
type EnclosureDTO struct {
	Animals []AnimalDTO `json:"animals"`
}

// UnplacedDTO pairs an un-placeable animal with a short reason code.
type UnplacedDTO struct {
	Animal AnimalDTO `json:"animal"`
	Reason string    `json:"reason"`
}

// AnimalDTO is the wire shape of a single animal.
type AnimalDTO struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Popularity     int      `json:"popularity"`
	MaximumFriends int      `json:"maximumFriends"`
	Incompatible   []string `json:"incompatible"`
}

// ZooFromAssignment converts an application-layer Assignment into its wire
// shape. Nil-safe in both directions: empty Enclosures/Unplaced render as
// [] rather than null.
func ZooFromAssignment(in appzoo.Assignment) ZooResponse {
	enc := make([]EnclosureDTO, len(in.Enclosures))
	for i, e := range in.Enclosures {
		enc[i] = EnclosureDTO{Animals: animalsFromDomain(e.Residents())}
	}
	unp := make([]UnplacedDTO, len(in.Unplaced))
	for i, u := range in.Unplaced {
		unp[i] = UnplacedDTO{
			Animal: animalDTO(u.Animal),
			Reason: u.Reason,
		}
	}
	return ZooResponse{Enclosures: enc, Unplaced: unp}
}

// ZooFromDomain is retained for call sites that have only the enclosure
// slice (e.g. legacy tests). It produces an empty Unplaced slice.
func ZooFromDomain(in []domzoo.Enclosure) ZooResponse {
	return ZooFromAssignment(appzoo.Assignment{Enclosures: in})
}

func animalsFromDomain(in []animal.Animal) []AnimalDTO {
	if len(in) == 0 {
		return []AnimalDTO{}
	}
	out := make([]AnimalDTO, len(in))
	for i, a := range in {
		out[i] = animalDTO(a)
	}
	return out
}

func animalDTO(a animal.Animal) AnimalDTO {
	return AnimalDTO{
		ID:             a.ID,
		Type:           a.Type,
		Popularity:     a.Popularity,
		MaximumFriends: a.MaximumFriends,
		Incompatible:   a.Incompatible,
	}
}
