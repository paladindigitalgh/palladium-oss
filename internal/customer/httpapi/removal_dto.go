package httpapi

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/customer/removal"
)

// removalEquipmentResponse is the JSON representation of one
// removal.EquipmentPreview entry.
type removalEquipmentResponse struct {
	ServiceEquipmentID uuid.UUID `json:"service_equipment_id"`
	DeviceID           uuid.UUID `json:"device_id"`
	DeviceName         string    `json:"device_name"`
	Role               string    `json:"role"`
	WillRunOLTTeardown bool      `json:"will_run_olt_teardown"`
}

// removalServiceResponse is the JSON representation of one
// removal.ServicePreview entry.
type removalServiceResponse struct {
	ServiceID   uuid.UUID                  `json:"service_id"`
	Description string                     `json:"description"`
	Status      string                     `json:"status"`
	Equipment   []removalEquipmentResponse `json:"equipment"`
}

// removalLocationResponse is the JSON representation of one
// removal.LocationPreview entry.
type removalLocationResponse struct {
	LocationID uuid.UUID                `json:"location_id"`
	Name       string                   `json:"name"`
	Status     string                   `json:"status"`
	Services   []removalServiceResponse `json:"services"`
}

// removalPreviewResponse is the JSON representation of removal.Preview.
type removalPreviewResponse struct {
	CustomerID uuid.UUID                 `json:"customer_id"`
	Locations  []removalLocationResponse `json:"locations"`
}

func newRemovalPreviewResponse(p removal.Preview) removalPreviewResponse {
	resp := removalPreviewResponse{CustomerID: p.CustomerID, Locations: make([]removalLocationResponse, len(p.Locations))}
	for i, loc := range p.Locations {
		locResp := removalLocationResponse{
			LocationID: loc.LocationID,
			Name:       loc.Name,
			Status:     string(loc.Status),
			Services:   make([]removalServiceResponse, len(loc.Services)),
		}
		for j, svc := range loc.Services {
			svcResp := removalServiceResponse{
				ServiceID:   svc.ServiceID,
				Description: svc.Description,
				Status:      string(svc.Status),
				Equipment:   make([]removalEquipmentResponse, len(svc.Equipment)),
			}
			for k, eq := range svc.Equipment {
				svcResp.Equipment[k] = removalEquipmentResponse{
					ServiceEquipmentID: eq.ServiceEquipmentID,
					DeviceID:           eq.DeviceID,
					DeviceName:         eq.DeviceName,
					Role:               string(eq.Role),
					WillRunOLTTeardown: eq.WillRunOLTTeardown,
				}
			}
			locResp.Services[j] = svcResp
		}
		resp.Locations[i] = locResp
	}
	return resp
}
