import type { OLT } from '@/types/olt'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real OLT data source. GET /olts has no server-side filtering (see
 * internal/olt/httpapi), so "an access network's OLTs" is resolved by
 * fetching every OLT once and filtering client-side -- the same pattern
 * locationRepository.ts uses. Unlike Location, OLT has its own Detail
 * page, so getOLTById hits GET /olts/:id directly (like
 * getCustomerById/getServiceById/getDeviceById) rather than fetching the
 * whole list and finding.
 */

interface OLTDto {
  id: string
  access_network_id: string
  name: string
  vendor: OLT['vendor']
  model: string
  management_ip_address: string
  connection_profile_id: string | null
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: OLTDto): OLT {
  return {
    id: dto.id,
    accessNetworkId: dto.access_network_id,
    name: dto.name,
    vendor: dto.vendor,
    model: dto.model,
    managementIpAddress: dto.management_ip_address,
    connectionProfileId: dto.connection_profile_id,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listOLTs(): Promise<OLT[]> {
  const { olts } = await apiFetch<{ olts: OLTDto[] }>('/olts/')
  return olts.map(fromDto)
}

export async function listOLTsByAccessNetworkId(accessNetworkId: string): Promise<OLT[]> {
  const olts = await listOLTs()
  return olts.filter((olt) => olt.accessNetworkId === accessNetworkId)
}

/** Fetches a single OLT, returning null (not throwing) when it does not exist. */
export async function getOLTById(id: string): Promise<OLT | null> {
  try {
    const dto = await apiFetch<OLTDto>(`/olts/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateOLTInput {
  accessNetworkId: string
  name: string
  vendor: OLT['vendor']
  model: string
  managementIpAddress: string
  description: string
}

/**
 * Creates an OLT. connectionProfileId is always sent as null -- there is
 * no picker for it in this workspace yet (see OLTFormDialog.vue); an OLT
 * can be created without one and have it set later once that UI exists.
 */
export async function createOLT(input: CreateOLTInput): Promise<OLT> {
  const dto = await apiFetch<OLTDto>('/olts/', {
    method: 'POST',
    body: {
      access_network_id: input.accessNetworkId,
      name: input.name,
      vendor: input.vendor,
      model: input.model,
      management_ip_address: input.managementIpAddress,
      description: input.description,
      connection_profile_id: null,
    },
  })
  return fromDto(dto)
}

export interface UpdateOLTInput {
  name: string
  vendor: OLT['vendor']
  model: string
  managementIpAddress: string
  description: string
  /**
   * Not user-editable (see OLTFormDialog.vue -- no accessNetwork picker
   * or connectionProfile picker exist in this workspace yet). Callers
   * pass the OLT's current accessNetworkId/connectionProfileId through
   * unchanged: PUT replaces every mutable column (see
   * internal/olt/postgres's Update), so omitting them here would
   * silently move the OLT to no access network and unassign a real
   * connection profile.
   */
  accessNetworkId: string
  connectionProfileId: string | null
}

export async function updateOLT(id: string, input: UpdateOLTInput): Promise<OLT> {
  const dto = await apiFetch<OLTDto>(`/olts/${id}`, {
    method: 'PUT',
    body: {
      access_network_id: input.accessNetworkId,
      name: input.name,
      vendor: input.vendor,
      model: input.model,
      management_ip_address: input.managementIpAddress,
      description: input.description,
      connection_profile_id: input.connectionProfileId,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the OLT identified by id. olts.id is referenced by
 * pon_ports.olt_id ON DELETE RESTRICT, so this throws an ApiError with
 * kind "conflict" if the OLT still has any PON Port.
 */
export async function deleteOLT(id: string): Promise<void> {
  await apiFetch<void>(`/olts/${id}`, { method: 'DELETE' })
}
