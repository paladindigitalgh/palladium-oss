import type { CustomerDevice } from '@/types/customerDevice'
import { apiFetch } from '@/services/api/httpClient'

interface CustomerDeviceDto {
  id: string
  customer_id: string
  device_id: string
  description: string
  attached_at: string | null
  detached_at: string | null
}

function fromDto(dto: CustomerDeviceDto): CustomerDevice {
  return {
    id: dto.id,
    customerId: dto.customer_id,
    deviceId: dto.device_id,
    description: dto.description,
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
  description: string
}

/**
 * Attaches an existing Device to a Customer, creating a CustomerDevice
 * record -- placing it at that Customer's premises independent of any
 * Service. attachedAt is always sent as "now" and detachedAt as null --
 * this only ever creates a fresh, currently-active attachment. The
 * backend rejects a Device that is already attached elsewhere, or that
 * is Retired, with an error (see
 * internal/customerdevice/service.CustomerDeviceService.Create).
 */
export async function attachCustomerDevice(input: AttachCustomerDeviceInput): Promise<CustomerDevice> {
  const dto = await apiFetch<CustomerDeviceDto>('/customer-devices/', {
    method: 'POST',
    body: {
      customer_id: input.customerId,
      device_id: input.deviceId,
      description: input.description,
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
      description: record.description,
      attached_at: record.attachedAt,
      detached_at: new Date().toISOString(),
    },
  })
  return fromDto(dto)
}
