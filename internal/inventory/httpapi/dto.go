// Package httpapi is the Inventory domain's REST layer. It depends on
// internal/inventory/service, never on a repository directly (see this
// milestone's goal 1), and never exposes internal/inventory's domain
// types over the wire — see the DTOs in this file.
//
// Site and Device are implemented here. Building and Room follow the
// same pattern once their own endpoints exist.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

// siteRequest is the JSON body for POST /api/v1/sites and
// PUT /api/v1/sites/{id}.
//
// It intentionally has no ID or timestamp fields. Identity is either
// server-assigned (POST) or comes from the URL path (PUT); CreatedAt and
// UpdatedAt are metadata the repository owns and a caller cannot set (see
// SiteRepository.Create's doc comment in
// internal/inventory/postgres/site.go for the same rule one layer down).
// Keeping the request DTO separate from inventory.Site — rather than
// exposing the domain model directly, per goal 4 — is what makes that
// rule enforceable here: there is no CreatedAt field on this struct for a
// client to even attempt to set.
type siteRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// toSite converts a request into a domain inventory.Site. id is supplied
// by the caller: uuid.Nil for Create (the repository assigns a real one),
// or the URL path parameter's UUID for Update.
func (req siteRequest) toSite(id uuid.UUID) inventory.Site {
	return inventory.Site{
		Metadata: inventory.Metadata{
			ID:          id,
			Name:        req.Name,
			Description: req.Description,
		},
	}
}

// siteResponse is the JSON representation of a Site returned to clients.
// Decoupling the wire format from inventory.Site's Go field layout (e.g.
// its embedded Metadata struct) means a change to how the domain model is
// composed internally can never silently change the API's JSON shape.
type siteResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newSiteResponse(site inventory.Site) siteResponse {
	return siteResponse{
		ID:          site.ID,
		Name:        site.Name,
		Description: site.Description,
		CreatedAt:   site.CreatedAt,
		UpdatedAt:   site.UpdatedAt,
	}
}

// siteListResponse wraps a slice of sites in an object rather than
// returning a bare JSON array. This is deliberate even though nothing
// needs it yet: a bare top-level array can never gain sibling fields
// (a total count, a pagination cursor, ...) without becoming a breaking
// change for existing clients, while adding a field next to "sites" is
// not.
type siteListResponse struct {
	Sites []siteResponse `json:"sites"`
}

func newSiteListResponse(sites []inventory.Site) siteListResponse {
	resp := siteListResponse{Sites: make([]siteResponse, len(sites))}
	for i, s := range sites {
		resp.Sites[i] = newSiteResponse(s)
	}
	return resp
}

// deviceRequest is the JSON body for POST /api/v1/devices and
// PUT /api/v1/devices/{id}. See siteRequest for why there is no ID or
// timestamp field.
type deviceRequest struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	RackID       *uuid.UUID `json:"rack_id"`
	Manufacturer string     `json:"manufacturer"`
	Model        string     `json:"model"`
	SerialNumber string     `json:"serial_number"`
	AssetTag     string     `json:"asset_tag"`
	Status       string     `json:"status"`
}

// toDevice converts a request into a domain inventory.Device. id is
// supplied by the caller, the same way siteRequest.toSite's is.
func (req deviceRequest) toDevice(id uuid.UUID) inventory.Device {
	return inventory.Device{
		Metadata: inventory.Metadata{
			ID:          id,
			Name:        req.Name,
			Description: req.Description,
		},
		RackID:       req.RackID,
		Manufacturer: req.Manufacturer,
		Model:        req.Model,
		SerialNumber: req.SerialNumber,
		AssetTag:     req.AssetTag,
		Status:       inventory.DeviceStatus(req.Status),
	}
}

// deviceResponse is the JSON representation of a Device returned to
// clients. See siteResponse for why this is a separate type from
// inventory.Device rather than the domain model exposed directly.
type deviceResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	RackID       *uuid.UUID `json:"rack_id"`
	Manufacturer string     `json:"manufacturer"`
	Model        string     `json:"model"`
	SerialNumber string     `json:"serial_number"`
	AssetTag     string     `json:"asset_tag"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func newDeviceResponse(device inventory.Device) deviceResponse {
	return deviceResponse{
		ID:           device.ID,
		Name:         device.Name,
		Description:  device.Description,
		RackID:       device.RackID,
		Manufacturer: device.Manufacturer,
		Model:        device.Model,
		SerialNumber: device.SerialNumber,
		AssetTag:     device.AssetTag,
		Status:       string(device.Status),
		CreatedAt:    device.CreatedAt,
		UpdatedAt:    device.UpdatedAt,
	}
}

// deviceListResponse wraps a slice of devices in an object rather than
// returning a bare JSON array. See siteListResponse for why.
type deviceListResponse struct {
	Devices []deviceResponse `json:"devices"`
}

func newDeviceListResponse(devices []inventory.Device) deviceListResponse {
	resp := deviceListResponse{Devices: make([]deviceResponse, len(devices))}
	for i, d := range devices {
		resp.Devices[i] = newDeviceResponse(d)
	}
	return resp
}
