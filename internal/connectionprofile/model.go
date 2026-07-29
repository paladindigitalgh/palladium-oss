// Package connectionprofile models Palladium's Connection Profile domain
// (v1): a named, reusable set of connection parameters — protocol, port,
// timeout, host-key verification policy, and which Authentication (see
// internal/authentication) to log in with — that a future device
// (an OLT, to start; see internal/olt's ConnectionProfileID) references
// to describe how to reach and log in to it. This package holds only the
// domain model, field validation, and the repository interface — no
// SQL, no migrations, no HTTP CRUD — mirroring internal/catalog's own
// package exactly.
//
// This is infrastructure, not a feature — the same distinction this
// milestone's own instructions draw explicitly for internal/authentication
// applies equally here ("This is NOT SSH execution. This is NOT OLT
// connectivity."). A ConnectionProfile record only describes how a
// future connection attempt would be configured; nothing in this
// package opens one.
//
// This package does not import internal/authentication. AuthenticationID
// is a bare *uuid.UUID, not a reference to authentication.Authentication:
// the foreign key to authentication_methods(id) is a database concept,
// enforced by internal/connectionprofile/postgres and its migration, not
// a Go package dependency — the same reasoning internal/olt/model.go
// documents for why OLT does not import internal/accessnetwork.
//
// # What this package deliberately does not validate
//
// This milestone's Rules section for ConnectionProfile lists exactly
// one requirement: "Name unique." Protocol, Port, AuthenticationID,
// Timeout, and Description are all left unvalidated beyond what their
// types already guarantee — see validate.go's own doc comment for the
// full reasoning, and HostKeyPolicy.go for the one field that is a
// closed enum (and is required, the same as every other closed-enum
// field in this codebase) despite not being called out by name in the
// Rules section either.
package connectionprofile

import (
	"time"

	"github.com/google/uuid"
)

// ConnectionProfile is a named, reusable set of connection parameters.
//
// AuthenticationID is nullable (*uuid.UUID), for the same reason
// inventory.Rack.RoomID is: this milestone's Rules section does not
// require it, so a ConnectionProfile can exist — e.g. as a template
// ("standard SSH, strict host-key checking, 30s timeout") — before any
// specific Authentication is bound to it.
//
// Timeout is a time.Duration, not an int, so its unit is never
// ambiguous in Go code — the same reasoning
// internal/platform/ssh.Config.Timeout is typed identically; a future
// caller building an ssh.Config from a ConnectionProfile passes this
// field straight through with no unit conversion to get wrong.
type ConnectionProfile struct {
	ID               uuid.UUID
	Name             string
	Protocol         string
	Port             int
	AuthenticationID *uuid.UUID
	Timeout          time.Duration
	HostKeyPolicy    HostKeyPolicy
	Description      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
