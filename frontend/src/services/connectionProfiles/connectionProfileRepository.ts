import type { ConnectionProfile } from '@/types/connectionProfile'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real ConnectionProfile data source. GET /connection-profiles has
 * no server-side filtering (see internal/connectionprofile/httpapi), the
 * same as oltModelRepository.ts's listOLTModels. Only the list is
 * exposed here -- the OLT Connection Profile picker in OLTFormDialog.vue
 * is this function's only caller, and it needs the full set, not any
 * one profile by id.
 */

interface ConnectionProfileDto {
  id: string
  name: string
  protocol: string
  port: number
  authentication_id: string | null
  timeout: string
  host_key_policy: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: ConnectionProfileDto): ConnectionProfile {
  return {
    id: dto.id,
    name: dto.name,
    protocol: dto.protocol,
    port: dto.port,
    authenticationId: dto.authentication_id,
    timeout: dto.timeout,
    hostKeyPolicy: dto.host_key_policy,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listConnectionProfiles(): Promise<ConnectionProfile[]> {
  const { connection_profiles: connectionProfiles } = await apiFetch<{ connection_profiles: ConnectionProfileDto[] }>(
    '/connection-profiles/',
  )
  return connectionProfiles.map(fromDto)
}
