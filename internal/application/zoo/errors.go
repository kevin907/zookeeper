package zoo

import "errors"

// Application-layer errors surfaced to the delivery layer via httperr.Map.
var (
	// ErrEnclosuresOutOfRange is returned when the requested enclosure count
	// is outside the supported [1, 8] range.
	ErrEnclosuresOutOfRange = errors.New("enclosures must be between 1 and 8")

	// ErrInvalidEnclosuresInput is returned when the enclosures path
	// parameter is not a parseable integer.
	ErrInvalidEnclosuresInput = errors.New("enclosures must be an integer")

	// ErrInfeasibleAssignment is returned when no valid assignment exists for
	// the given roster and enclosure count.
	ErrInfeasibleAssignment = errors.New("no feasible enclosure assignment")

	// ErrDependencyUnavailable is returned when a downstream dependency
	// (e.g. Postgres) fails a readiness probe.
	ErrDependencyUnavailable = errors.New("dependency unavailable")
)
