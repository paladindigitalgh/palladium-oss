package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accesstopology"
	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
)

// customerResolver is the seam AccessTopologyHandler depends on instead
// of a concrete *accesstopology.CustomerResolver — the same reasoning
// every other domain's *service interface in this codebase documents:
// it lets handler tests exercise HTTP behavior against a fake, with no
// real resolver or repositories involved.
//
// There is no separate service layer here, unlike most domains'
// httpapi packages: accesstopology.CustomerResolver.LocateForCustomer
// already is the business logic (validate-then-delegate has nothing to
// add — there is no input to validate beyond the path parameter, and no
// state to persist), the same reasoning internal/event's handler depends
// on its repository directly (see cmd/server/main.go's own comment on
// why Event has no service layer).
type customerResolver interface {
	LocateForCustomer(ctx context.Context, customerID uuid.UUID) ([]accesstopology.CustomerLocation, error)
}

// AccessTopologyHandler serves the Access Topology domain's one REST
// endpoint:
//
//	GET /api/v1/diagnostics/customers/{customerId}/equipment-locations
//
// This is a GET, unlike every endpoint in
// internal/diagnostics/kontron/httpapi: those run a real command against
// live hardware (POST, per that package's own doc comment); this one
// only reads Palladium's own database — no device is ever contacted —
// so a plain, cacheable-in-spirit GET is the honest verb.
type AccessTopologyHandler struct {
	resolver customerResolver
}

// NewAccessTopologyHandler builds an AccessTopologyHandler.
func NewAccessTopologyHandler(resolver customerResolver) *AccessTopologyHandler {
	return &AccessTopologyHandler{resolver: resolver}
}

// ListCustomerEquipmentLocations handles GET
// /api/v1/diagnostics/customers/{customerId}/equipment-locations.
func (h *AccessTopologyHandler) ListCustomerEquipmentLocations(w http.ResponseWriter, r *http.Request) {
	customerID, err := pathCustomerID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	locations, err := h.resolver.LocateForCustomer(r.Context(), customerID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCustomerLocationsResponse(locations))
}

func pathCustomerID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "customerId"))
	if err != nil {
		return uuid.Nil, apperror.Invalid("customerId must be a valid UUID")
	}
	return id, nil
}
