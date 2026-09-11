import type { CustomerContactRow, CustomerDeviceRow, DeviceReportRow } from '@/types/report'
import { apiFetch } from '@/services/api/httpClient'

/**
 * Explorer's three curated reports (internal/report;
 * docs/09-WORKSPACE-SPECIFICATIONS.md §15). Unlike every other
 * `*Repository.ts` in this codebase, there is no per-row filter/sort/
 * pagination query here at all: a report is fetched once, in full, when
 * an operator picks it in ExplorerReportsView.vue, and every reshaping
 * after that (search, sort, pagination, CSV export) happens entirely
 * client-side against that one fetched array -- see
 * ExplorerReportsView.vue's own doc comment for why that is a deliberate
 * departure from deviceRepository.ts's "refetch on every filter change"
 * shape, not an inconsistency.
 */

// NIL_UUID is the wire form of Go's uuid.Nil, exactly as
// internal/report/postgres/report.go's COALESCE(..., '00000000-...')
// produces it for "no Contact"/"no assigned Customer" -- the backend's
// literal sentinel for absence, translated to `null` here at the one
// boundary that needs to know about it, so nothing downstream carries
// magic-string knowledge of it.
const NIL_UUID = '00000000-0000-0000-0000-000000000000'

function nullableId(raw: string): string | null {
  return raw === NIL_UUID ? null : raw
}

interface CustomerContactRowDto {
  customer_id: string
  customer_name: string
  customer_type: string
  customer_status: string
  contact_id: string
  contact_name: string
  contact_role: string
  contact_email: string
  contact_phone: string
  contact_status: string
}

function fromCustomerContactDto(dto: CustomerContactRowDto): CustomerContactRow {
  return {
    customerId: dto.customer_id,
    customerName: dto.customer_name,
    customerType: dto.customer_type,
    customerStatus: dto.customer_status,
    contactId: nullableId(dto.contact_id),
    contactName: dto.contact_name,
    contactRole: dto.contact_role,
    contactEmail: dto.contact_email,
    contactPhone: dto.contact_phone,
    contactStatus: dto.contact_status,
  }
}

/** Fetches the "Customers & Contacts" report in full. */
export async function listCustomersWithContacts(): Promise<CustomerContactRow[]> {
  const { rows } = await apiFetch<{ rows: CustomerContactRowDto[] }>('/reports/customers-contacts')
  return rows.map(fromCustomerContactDto)
}

interface DeviceReportRowDto {
  device_id: string
  device_name: string
  serial_number: string
  asset_tag: string
  device_status: string
  manufacturer: string
  model: string
  site_name: string
  building_name: string
  room_name: string
  rack_name: string
  assigned_customer_id: string
  assigned_customer_name: string
}

function fromDeviceReportDto(dto: DeviceReportRowDto): DeviceReportRow {
  return {
    deviceId: dto.device_id,
    deviceName: dto.device_name,
    serialNumber: dto.serial_number,
    assetTag: dto.asset_tag,
    deviceStatus: dto.device_status,
    manufacturer: dto.manufacturer,
    model: dto.model,
    siteName: dto.site_name,
    buildingName: dto.building_name,
    roomName: dto.room_name,
    rackName: dto.rack_name,
    assignedCustomerId: nullableId(dto.assigned_customer_id),
    assignedCustomerName: dto.assigned_customer_name,
  }
}

/** Fetches the "Devices" report in full. */
export async function listDevicesReport(): Promise<DeviceReportRow[]> {
  const { rows } = await apiFetch<{ rows: DeviceReportRowDto[] }>('/reports/devices')
  return rows.map(fromDeviceReportDto)
}

interface CustomerDeviceRowDto {
  customer_id: string
  customer_name: string
  customer_type: string
  customer_status: string
  device_id: string
  device_name: string
  serial_number: string
  manufacturer: string
  model: string
  device_status: string
  relationship: CustomerDeviceRow['relationship']
  service_status: string
  location_name: string
}

function fromCustomerDeviceDto(dto: CustomerDeviceRowDto): CustomerDeviceRow {
  return {
    customerId: dto.customer_id,
    customerName: dto.customer_name,
    customerType: dto.customer_type,
    customerStatus: dto.customer_status,
    deviceId: dto.device_id,
    deviceName: dto.device_name,
    serialNumber: dto.serial_number,
    manufacturer: dto.manufacturer,
    model: dto.model,
    deviceStatus: dto.device_status,
    relationship: dto.relationship,
    serviceStatus: dto.service_status,
    locationName: dto.location_name,
  }
}

/** Fetches the "Customers & Devices" report in full. */
export async function listCustomersWithDevices(): Promise<CustomerDeviceRow[]> {
  const { rows } = await apiFetch<{ rows: CustomerDeviceRowDto[] }>('/reports/customers-devices')
  return rows.map(fromCustomerDeviceDto)
}
