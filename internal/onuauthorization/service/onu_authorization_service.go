// Package service is the ONU Authorization domain's business logic
// layer, mirroring internal/accessattachment/service exactly, minus that
// package's active-attachment-uniqueness enforcement: OnuAuthorization
// has the same "at most one active record per Device" shape, but nothing
// in this codebase creates a second OnuAuthorization for an
// already-actively-authorized Device today (see
// internal/provisioning/kontron/service.AuthorizeAndCreateDeviceService,
// this domain's one writer), so there is no real call site that could
// violate it yet — adding the check now would be speculative, not
// load-bearing (CLAUDE.md: "avoid unnecessary abstractions").
package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/onuauthorization"
)

// OnuAuthorizationService is the ONU Authorization domain's business
// logic. It depends only on onuauthorization.Repository, the same
// reasoning internal/accessattachment/service.AccessAttachmentService
// documents for depending on its own repository alone: timestamps are
// already the repository's responsibility.
type OnuAuthorizationService struct {
	authorizations onuauthorization.Repository
}

// NewOnuAuthorizationService builds an OnuAuthorizationService.
func NewOnuAuthorizationService(authorizations onuauthorization.Repository) *OnuAuthorizationService {
	return &OnuAuthorizationService{authorizations: authorizations}
}

// Create validates authorization and, if valid, persists it.
func (s *OnuAuthorizationService) Create(ctx context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	if err := authorization.Validate(); err != nil {
		return onuauthorization.OnuAuthorization{}, err
	}
	return s.authorizations.Create(ctx, authorization)
}

// Update validates authorization and, if valid, persists the change.
func (s *OnuAuthorizationService) Update(ctx context.Context, authorization onuauthorization.OnuAuthorization) (onuauthorization.OnuAuthorization, error) {
	if err := authorization.Validate(); err != nil {
		return onuauthorization.OnuAuthorization{}, err
	}
	return s.authorizations.Update(ctx, authorization)
}

// GetActiveByDeviceID retrieves the active OnuAuthorization for
// deviceID, or an apperror.KindNotFound error if it has none.
func (s *OnuAuthorizationService) GetActiveByDeviceID(ctx context.Context, deviceID uuid.UUID) (onuauthorization.OnuAuthorization, error) {
	return s.authorizations.GetActiveByDeviceID(ctx, deviceID)
}
