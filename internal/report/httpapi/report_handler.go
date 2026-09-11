package httpapi

import (
	"context"
	"net/http"

	"github.com/paladindigitalgh/palladium-oss/internal/httpx"
	"github.com/paladindigitalgh/palladium-oss/internal/report"
)

// reportRepository is the seam ReportHandler depends on instead of a
// concrete report.Repository, so handler tests can exercise HTTP
// behavior against a fake.
type reportRepository interface {
	CustomersWithContacts(ctx context.Context) ([]report.CustomerContactRow, error)
	Devices(ctx context.Context) ([]report.DeviceRow, error)
	CustomersWithDevices(ctx context.Context) ([]report.CustomerDeviceRow, error)
}

// ReportHandler serves Explorer's three curated reports:
//
//	GET /api/v1/reports/customers-contacts
//	GET /api/v1/reports/devices
//	GET /api/v1/reports/customers-devices
//
// Every route this handler serves is mounted behind RequireReports
// alone (see internal/server/router.go) — every Role can read every
// report, the same as the underlying Customer/Device/Contact data each
// one already exposes individually. There is no business logic here:
// each method is a thin decode/delegate/translate, exactly like every
// other httpapi package in this codebase — the actual query lives in
// internal/report/postgres.
type ReportHandler struct {
	reports reportRepository
}

// NewReportHandler builds a ReportHandler.
func NewReportHandler(reports reportRepository) *ReportHandler {
	return &ReportHandler{reports: reports}
}

// CustomersWithContacts handles GET /api/v1/reports/customers-contacts.
func (h *ReportHandler) CustomersWithContacts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.reports.CustomersWithContacts(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newCustomersWithContactsResponse(rows))
}

// Devices handles GET /api/v1/reports/devices.
func (h *ReportHandler) Devices(w http.ResponseWriter, r *http.Request) {
	rows, err := h.reports.Devices(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newDevicesReportResponse(rows))
}

// CustomersWithDevices handles GET /api/v1/reports/customers-devices.
func (h *ReportHandler) CustomersWithDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := h.reports.CustomersWithDevices(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newCustomersWithDevicesResponse(rows))
}
