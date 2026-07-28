// Package httpapi is the Access Attachment domain's REST layer. It
// depends on internal/accessattachment/service, never on a repository
// directly, and never exposes internal/accessattachment's domain types
// over the wire — see the DTOs in this file. It mirrors
// internal/serviceequipment/httpapi exactly.
package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/accessattachment"
)

// accessAttachmentRequest is the JSON body for POST
// /api/v1/access-attachments and PUT /api/v1/access-attachments/{id}.
//
// AccessInterfaceID, ServiceEquipmentID, and the two lifecycle
// timestamps are left as their plain primitive types (uuid.UUID,
// *time.Time) — the same reasoning
// internal/serviceequipment/httpapi.serviceEquipmentRequest gives for
// its own ServiceID/DeviceID/InstalledAt/RemovedAt fields: they carry no
// domain enum type to decouple from in the first place.
//
// It intentionally has no ID or CreatedAt/UpdatedAt fields. Identity is
// either server-assigned (POST) or comes from the URL path (PUT);
// CreatedAt and UpdatedAt are metadata the repository owns and a caller
// cannot set.
type accessAttachmentRequest struct {
	AccessInterfaceID  uuid.UUID `json:"access_interface_id"`
	ServiceEquipmentID uuid.UUID `json:"service_equipment_id"`
	RemovalReason      string    `json:"removal_reason"`

	InstalledAt *time.Time `json:"installed_at"`
	RemovedAt   *time.Time `json:"removed_at"`
}

// toAccessAttachment converts a request into a domain AccessAttachment.
// id is supplied by the caller: uuid.Nil for Create (the repository
// assigns a real one), or the URL path parameter's UUID for Update.
func (req accessAttachmentRequest) toAccessAttachment(id uuid.UUID) accessattachment.AccessAttachment {
	return accessattachment.AccessAttachment{
		ID:                 id,
		AccessInterfaceID:  req.AccessInterfaceID,
		ServiceEquipmentID: req.ServiceEquipmentID,
		RemovalReason:      req.RemovalReason,

		InstalledAt: req.InstalledAt,
		RemovedAt:   req.RemovedAt,
	}
}

// accessAttachmentResponse is the JSON representation of an
// AccessAttachment returned to clients. Decoupling the wire format from
// AccessAttachment's Go field layout and types means a change to how the
// domain model is composed internally can never silently change the
// API's JSON shape.
type accessAttachmentResponse struct {
	ID                 uuid.UUID `json:"id"`
	AccessInterfaceID  uuid.UUID `json:"access_interface_id"`
	ServiceEquipmentID uuid.UUID `json:"service_equipment_id"`
	RemovalReason      string    `json:"removal_reason"`

	InstalledAt *time.Time `json:"installed_at"`
	RemovedAt   *time.Time `json:"removed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newAccessAttachmentResponse(a accessattachment.AccessAttachment) accessAttachmentResponse {
	return accessAttachmentResponse{
		ID:                 a.ID,
		AccessInterfaceID:  a.AccessInterfaceID,
		ServiceEquipmentID: a.ServiceEquipmentID,
		RemovalReason:      a.RemovalReason,

		InstalledAt: a.InstalledAt,
		RemovedAt:   a.RemovedAt,

		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// accessAttachmentListResponse wraps a slice of attachments in an object
// rather than returning a bare JSON array — the same reasoning as
// internal/serviceequipment/httpapi's serviceEquipmentListResponse: a
// bare top-level array can never gain sibling fields (a total count, a
// pagination cursor, ...) without becoming a breaking change for
// existing clients, while adding a field next to "access_attachments" is
// not.
type accessAttachmentListResponse struct {
	AccessAttachments []accessAttachmentResponse `json:"access_attachments"`
}

func newAccessAttachmentListResponse(attachments []accessattachment.AccessAttachment) accessAttachmentListResponse {
	resp := accessAttachmentListResponse{AccessAttachments: make([]accessAttachmentResponse, len(attachments))}
	for i, a := range attachments {
		resp.AccessAttachments[i] = newAccessAttachmentResponse(a)
	}
	return resp
}
