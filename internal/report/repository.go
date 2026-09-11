package report

import "context"

// Repository answers Explorer's three curated reports (see this
// package's own doc comment). There is deliberately no Create, Update,
// or Delete — every report is a read query over data other domains
// already own and write to; this package never mutates anything.
//
// Nothing in this package implements Repository — no SQL, no
// migrations — so the domain has zero dependency on any storage
// technology, the same separation every other *Repository interface in
// this codebase keeps. internal/report/postgres satisfies it.
type Repository interface {
	// CustomersWithContacts backs the "Customers & Contacts" report.
	CustomersWithContacts(ctx context.Context) ([]CustomerContactRow, error)

	// Devices backs the "Devices" report.
	Devices(ctx context.Context) ([]DeviceRow, error)

	// CustomersWithDevices backs the "Customers & Devices" report.
	CustomersWithDevices(ctx context.Context) ([]CustomerDeviceRow, error)
}
