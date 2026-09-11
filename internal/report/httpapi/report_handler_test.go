package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/report"
	"github.com/paladindigitalgh/palladium-oss/internal/report/httpapi"
)

// stubReportRepository is the seam httpapi.ReportHandler depends on. It
// lets these tests exercise HTTP-only concerns -- status codes, JSON
// shapes, error translation -- without a real repository or database;
// internal/report/postgres has its own integration tests for the SQL
// itself.
type stubReportRepository struct {
	contactRows []report.CustomerContactRow
	deviceRows  []report.DeviceRow
	joinRows    []report.CustomerDeviceRow
	err         error
}

func (s *stubReportRepository) CustomersWithContacts(context.Context) ([]report.CustomerContactRow, error) {
	return s.contactRows, s.err
}

func (s *stubReportRepository) Devices(context.Context) ([]report.DeviceRow, error) {
	return s.deviceRows, s.err
}

func (s *stubReportRepository) CustomersWithDevices(context.Context) ([]report.CustomerDeviceRow, error) {
	return s.joinRows, s.err
}

func TestCustomersWithContactsReturnsRows(t *testing.T) {
	stub := &stubReportRepository{contactRows: []report.CustomerContactRow{
		{CustomerID: uuid.New(), CustomerName: "Acme", CustomerType: "Business", CustomerStatus: "Active"},
	}}
	h := httpapi.NewReportHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customers-contacts", nil)
	rec := httptest.NewRecorder()
	h.CustomersWithContacts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Rows []struct {
			CustomerName string `json:"customer_name"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Rows) != 1 || body.Rows[0].CustomerName != "Acme" {
		t.Errorf("rows = %+v, want one row for Acme", body.Rows)
	}
}

func TestCustomersWithContactsPropagatesRepositoryError(t *testing.T) {
	h := httpapi.NewReportHandler(&stubReportRepository{err: apperror.Internal("boom", nil)})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customers-contacts", nil)
	rec := httptest.NewRecorder()
	h.CustomersWithContacts(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
}

func TestDevicesReturnsRows(t *testing.T) {
	stub := &stubReportRepository{deviceRows: []report.DeviceRow{
		{DeviceID: uuid.New(), DeviceName: "OLT-01", SerialNumber: "SN1"},
	}}
	h := httpapi.NewReportHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/devices", nil)
	rec := httptest.NewRecorder()
	h.Devices(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Rows []struct {
			DeviceName string `json:"device_name"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Rows) != 1 || body.Rows[0].DeviceName != "OLT-01" {
		t.Errorf("rows = %+v, want one row for OLT-01", body.Rows)
	}
}

func TestCustomersWithDevicesReturnsRows(t *testing.T) {
	stub := &stubReportRepository{joinRows: []report.CustomerDeviceRow{
		{CustomerName: "Acme", DeviceName: "OLT-01", Relationship: "Placement"},
	}}
	h := httpapi.NewReportHandler(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customers-devices", nil)
	rec := httptest.NewRecorder()
	h.CustomersWithDevices(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		Rows []struct {
			Relationship string `json:"relationship"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Rows) != 1 || body.Rows[0].Relationship != "Placement" {
		t.Errorf("rows = %+v, want one Placement row", body.Rows)
	}
}
