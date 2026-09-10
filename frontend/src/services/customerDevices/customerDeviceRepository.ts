import type { CustomerDevice } from '@/types/customerDevice'
import { apiFetch } from '@/services/api/httpClient'

interface CustomerDeviceDto {
  id: string
  customer_id: string
  device_id: string
  location_id: string | null
  attached_at: string | null
  detached_at: string | null
}

function fromDto(dto: CustomerDeviceDto): CustomerDevice {
  return {
    id: dto.id,
    customerId: dto.customer_id,
    deviceId: dto.device_id,
    locationId: dto.location_id,
    attachedAt: dto.attached_at,
    detachedAt: dto.detached_at,
  }
}

/**
 * GET /customer-devices has no server-side filtering (see
 * internal/customerdevice/httpapi), so every list below fetches the full
 * set once and filters client-side -- the same pattern
 * serviceEquipmentRepository.ts's own listServiceEquipment/
 * listServiceEquipmentByDeviceId establish.
 */
export async function listCustomerDevices(): Promise<CustomerDevice[]> {
  const { customer_devices: records } = await apiFetch<{ customer_devices: CustomerDeviceDto[] }>('/customer-devices/')
  return records.map(fromDto)
}

/** Every CustomerDevice record (active or historical) for one Customer. */
export async function listCustomerDevicesByCustomerId(customerId: string): Promise<CustomerDevice[]> {
  const records = await listCustomerDevices()
  return records.filter((r) => r.customerId === customerId)
}

/** Just the Customer's currently-attached devices (detachedAt === null). */
export async function listActiveCustomerDevicesByCustomerId(customerId: string): Promise<CustomerDevice[]> {
  return (await listCustomerDevicesByCustomerId(customerId)).filter((r) => r.detachedAt === null)
}

export interface AttachCustomerDeviceInput {
  customerId: string
  deviceId: string
  /** Which of customerId's own Locations this Device physically sits at -- optional, purely for tracking (see CustomerDevice.locationId). */
  locationId: string | null
}

/**
 * Attaches an existing Device to a Customer, creating a CustomerDevice
 * record -- placing it at that Customer's premises independent of any
 * Service. attachedAt is always sent as "now" and detachedAt as null --
 * this only ever creates a fresh, currently-active attachment. The
 * backend rejects a Device that is already attached elsewhere, or that
 * is Retired, with an error (see
 * internal/customerdevice/service.CustomerDeviceService.Create) -- and,
 * if locationId is set, rejects one that does not belong to customerId
 * (see that method's own ensureLocationBelongsToCustomer check).
 */
export async function attachCustomerDevice(input: AttachCustomerDeviceInput): Promise<CustomerDevice> {
  const dto = await apiFetch<CustomerDeviceDto>('/customer-devices/', {
    method: 'POST',
    body: {
      customer_id: input.customerId,
      device_id: input.deviceId,
      location_id: input.locationId,
      attached_at: new Date().toISOString(),
      detached_at: null,
    },
  })
  return fromDto(dto)
}

/**
 * Detaches a Device from its Customer by setting detachedAt -- the
 * CustomerDevice row itself is preserved, never deleted (this domain has
 * no Delete at all; see internal/customerdevice.CustomerDeviceRepository's
 * own doc comment). Every other field is echoed back unchanged, the same
 * full-record PUT convention updateService/updateLocation use. The
 * backend rejects the detach with a conflict error while the Device
 * still fulfills an active Service (see
 * internal/customerdevice/service.CustomerDeviceService.Update) --
 * remove it from the Service first.
 */
export async function detachCustomerDevice(record: CustomerDevice): Promise<CustomerDevice> {
  const dto = await apiFetch<CustomerDeviceDto>(`/customer-devices/${record.id}`, {
    method: 'PUT',
    body: {
      customer_id: record.customerId,
      device_id: record.deviceId,
      location_id: record.locationId,
      attached_at: record.attachedAt,
      detached_at: new Date().toISOString(),
    },
  })
  return fromDto(dto)
}

/**
 * Sets or clears which of the Customer's own Locations a Device is
 * recorded as sitting at, on an already-attached CustomerDevice -- an
 * operator correcting or filling in tracking data after the fact,
 * without detaching and reattaching. Every other field is echoed back
 * unchanged, the same full-record PUT convention detachCustomerDevice
 * above uses. The backend rejects a Location that does not belong to
 * this record's own customerId (see
 * internal/customerdevice/service.CustomerDeviceService's
 * ensureLocationBelongsToCustomer).
 */
export async function setCustomerDeviceLocation(record: CustomerDevice, locationId: string | null): Promise<CustomerDevice> {
  const dto = await apiFetch<CustomerDeviceDto>(`/customer-devices/${record.id}`, {
    method: 'PUT',
    body: {
      customer_id: record.customerId,
      device_id: record.deviceId,
      location_id: locationId,
      attached_at: record.attachedAt,
      detached_at: record.detachedAt,
    },
  })
  return fromDto(dto)
}
