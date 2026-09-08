import type { CustomerRemovalPreview } from '@/types/customerRemoval'
import { apiFetch } from '@/services/api/httpClient'

interface CustomerRemovalEquipmentPreviewDto {
  service_equipment_id: string
  device_id: string
  device_name: string
  role: string
  will_run_olt_teardown: boolean
}

interface CustomerRemovalServicePreviewDto {
  service_id: string
  description: string
  status: string
  equipment: CustomerRemovalEquipmentPreviewDto[]
}

interface CustomerRemovalLocationPreviewDto {
  location_id: string
  name: string
  status: string
  services: CustomerRemovalServicePreviewDto[]
}

interface CustomerRemovalPreviewDto {
  customer_id: string
  locations: CustomerRemovalLocationPreviewDto[]
}

function fromDto(dto: CustomerRemovalPreviewDto): CustomerRemovalPreview {
  return {
    customerId: dto.customer_id,
    locations: dto.locations.map((loc) => ({
      locationId: loc.location_id,
      name: loc.name,
      status: loc.status,
      services: loc.services.map((svc) => ({
        serviceId: svc.service_id,
        description: svc.description,
        status: svc.status,
        equipment: svc.equipment.map((eq) => ({
          serviceEquipmentId: eq.service_equipment_id,
          deviceId: eq.device_id,
          deviceName: eq.device_name,
          role: eq.role,
          willRunOltTeardown: eq.will_run_olt_teardown,
        })),
      })),
    })),
  }
}

/**
 * Fetches what "Remove Customer" would do for id, without changing
 * anything -- see internal/customer/removal's own doc comment.
 */
export async function getCustomerRemovalPreview(id: string): Promise<CustomerRemovalPreview> {
  const dto = await apiFetch<CustomerRemovalPreviewDto>(`/customers/${id}/removal-preview`)
  return fromDto(dto)
}

/**
 * Executes "Remove Customer" for id: Locations go Inactive, Services go
 * Disconnected (running a real OLT teardown first when one currently
 * applies), and their equipment is unassigned -- the underlying Device
 * is never touched. Safe to call again if it fails partway through; it
 * picks up from wherever it stopped.
 */
export async function executeCustomerRemoval(id: string): Promise<void> {
  await apiFetch<void>(`/customers/${id}/removal`, { method: 'POST' })
}
