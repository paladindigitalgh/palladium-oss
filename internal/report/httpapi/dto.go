// Package httpapi is the Report domain's REST layer: three read-only
// GET routes backing Explorer's curated reports
// (docs/09-WORKSPACE-SPECIFICATIONS.md §15). There is no create/update
// route — see internal/report's own package doc comment for why.
package httpapi

import (
	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/report"
)

// customerContactRowResponse is the JSON representation of a
// report.CustomerContactRow, decoupled from its Go field layout.
type customerContactRowResponse struct {
	CustomerID     uuid.UUID `json:"customer_id"`
	CustomerName   string    `json:"customer_name"`
	CustomerType   string    `json:"customer_type"`
	CustomerStatus string    `json:"customer_status"`
	ContactID      uuid.UUID `json:"contact_id"`
	ContactName    string    `json:"contact_name"`
	ContactRole    string    `json:"contact_role"`
	ContactEmail   string    `json:"contact_email"`
	ContactPhone   string    `json:"contact_phone"`
	ContactStatus  string    `json:"contact_status"`
}

func newCustomerContactRowResponse(row report.CustomerContactRow) customerContactRowResponse {
	return customerContactRowResponse{
		CustomerID:     row.CustomerID,
		CustomerName:   row.CustomerName,
		CustomerType:   row.CustomerType,
		CustomerStatus: row.CustomerStatus,
		ContactID:      row.ContactID,
		ContactName:    row.ContactName,
		ContactRole:    row.ContactRole,
		ContactEmail:   row.ContactEmail,
		ContactPhone:   row.ContactPhone,
		ContactStatus:  row.ContactStatus,
	}
}

type customersWithContactsResponse struct {
	Rows []customerContactRowResponse `json:"rows"`
}

func newCustomersWithContactsResponse(rows []report.CustomerContactRow) customersWithContactsResponse {
	resp := customersWithContactsResponse{Rows: make([]customerContactRowResponse, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = newCustomerContactRowResponse(row)
	}
	return resp
}

// deviceRowResponse is the JSON representation of a report.DeviceRow.
type deviceRowResponse struct {
	DeviceID             uuid.UUID `json:"device_id"`
	DeviceName           string    `json:"device_name"`
	SerialNumber         string    `json:"serial_number"`
	AssetTag             string    `json:"asset_tag"`
	DeviceStatus         string    `json:"device_status"`
	Manufacturer         string    `json:"manufacturer"`
	Model                string    `json:"model"`
	SiteName             string    `json:"site_name"`
	BuildingName         string    `json:"building_name"`
	RoomName             string    `json:"room_name"`
	RackName             string    `json:"rack_name"`
	AssignedCustomerID   uuid.UUID `json:"assigned_customer_id"`
	AssignedCustomerName string    `json:"assigned_customer_name"`
}

func newDeviceRowResponse(row report.DeviceRow) deviceRowResponse {
	return deviceRowResponse{
		DeviceID:             row.DeviceID,
		DeviceName:           row.DeviceName,
		SerialNumber:         row.SerialNumber,
		AssetTag:             row.AssetTag,
		DeviceStatus:         row.DeviceStatus,
		Manufacturer:         row.Manufacturer,
		Model:                row.Model,
		SiteName:             row.SiteName,
		BuildingName:         row.BuildingName,
		RoomName:             row.RoomName,
		RackName:             row.RackName,
		AssignedCustomerID:   row.AssignedCustomerID,
		AssignedCustomerName: row.AssignedCustomerName,
	}
}

type devicesReportResponse struct {
	Rows []deviceRowResponse `json:"rows"`
}

func newDevicesReportResponse(rows []report.DeviceRow) devicesReportResponse {
	resp := devicesReportResponse{Rows: make([]deviceRowResponse, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = newDeviceRowResponse(row)
	}
	return resp
}

// customerDeviceRowResponse is the JSON representation of a
// report.CustomerDeviceRow.
type customerDeviceRowResponse struct {
	CustomerID     uuid.UUID `json:"customer_id"`
	CustomerName   string    `json:"customer_name"`
	CustomerType   string    `json:"customer_type"`
	CustomerStatus string    `json:"customer_status"`
	DeviceID       uuid.UUID `json:"device_id"`
	DeviceName     string    `json:"device_name"`
	SerialNumber   string    `json:"serial_number"`
	Manufacturer   string    `json:"manufacturer"`
	Model          string    `json:"model"`
	DeviceStatus   string    `json:"device_status"`
	Relationship   string    `json:"relationship"`
	ServiceStatus  string    `json:"service_status"`
	LocationName   string    `json:"location_name"`
}

func newCustomerDeviceRowResponse(row report.CustomerDeviceRow) customerDeviceRowResponse {
	return customerDeviceRowResponse{
		CustomerID:     row.CustomerID,
		CustomerName:   row.CustomerName,
		CustomerType:   row.CustomerType,
		CustomerStatus: row.CustomerStatus,
		DeviceID:       row.DeviceID,
		DeviceName:     row.DeviceName,
		SerialNumber:   row.SerialNumber,
		Manufacturer:   row.Manufacturer,
		Model:          row.Model,
		DeviceStatus:   row.DeviceStatus,
		Relationship:   row.Relationship,
		ServiceStatus:  row.ServiceStatus,
		LocationName:   row.LocationName,
	}
}

type customersWithDevicesResponse struct {
	Rows []customerDeviceRowResponse `json:"rows"`
}

func newCustomersWithDevicesResponse(rows []report.CustomerDeviceRow) customersWithDevicesResponse {
	resp := customersWithDevicesResponse{Rows: make([]customerDeviceRowResponse, len(rows))}
	for i, row := range rows {
		resp.Rows[i] = newCustomerDeviceRowResponse(row)
	}
	return resp
}
