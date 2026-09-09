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
  olt_model_id: string
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
    oltModelId: dto.olt_model_id,
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
  oltModelId: string
  managementIpAddress: string
  description: string
  connectionProfileId: string | null
}

/**
 * Creates an OLT. oltModelId is required (see internal/olt.OLT.OLTModelID)
 * -- creating this OLT auto-creates PON ports matching the chosen
 * model's port count, on the backend. connectionProfileId is nullable
 * (see OLTFormDialog.vue's picker): an OLT can still be created without
 * one and have it set later, e.g. before a Connection Profile exists to
 * assign.
 */
export async function createOLT(input: CreateOLTInput): Promise<OLT> {
  const dto = await apiFetch<OLTDto>('/olts/', {
    method: 'POST',
    body: {
      access_network_id: input.accessNetworkId,
      name: input.name,
      olt_model_id: input.oltModelId,
      management_ip_address: input.managementIpAddress,
      description: input.description,
      connection_profile_id: input.connectionProfileId,
    },
  })
  return fromDto(dto)
}

export interface UpdateOLTInput {
  name: string
  oltModelId: string
  managementIpAddress: string
  description: string
  connectionProfileId: string | null
  /**
   * Not user-editable (see OLTFormDialog.vue -- no accessNetwork picker
   * exists in this workspace). Callers pass the OLT's current
   * accessNetworkId through unchanged: PUT replaces every mutable column
   * (see internal/olt/postgres's Update), so omitting it here would
   * silently move the OLT to no access network.
   */
  accessNetworkId: string
}

export async function updateOLT(id: string, input: UpdateOLTInput): Promise<OLT> {
  const dto = await apiFetch<OLTDto>(`/olts/${id}`, {
    method: 'PUT',
    body: {
      access_network_id: input.accessNetworkId,
      name: input.name,
      olt_model_id: input.oltModelId,
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
