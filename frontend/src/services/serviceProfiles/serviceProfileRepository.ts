import type { ServiceProfile } from '@/types/serviceProfile'
import { apiFetch } from '@/services/api/httpClient'

/** Read-only: there is no Service Profile Workspace yet, only the Service creation form's dropdown. */

interface ServiceProfileDto {
  id: string
  name: string
  status: ServiceProfile['status']
}

export async function listServiceProfiles(): Promise<ServiceProfile[]> {
  const { service_profiles: profiles } = await apiFetch<{ service_profiles: ServiceProfileDto[] }>('/service-profiles/')
  return profiles.map((dto) => ({ id: dto.id, name: dto.name, status: dto.status }))
}
