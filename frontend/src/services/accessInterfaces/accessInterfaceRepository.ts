import type { AccessInterface } from '@/types/accessInterface'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real AccessInterface data source. Mirrors oltRepository.ts's shape
 * exactly -- GET /access-interfaces has no server-side filtering, so "a
 * PON Port's Access Interfaces" is resolved by fetching every Access
 * Interface once and filtering client-side, and getAccessInterfaceById
 * hits GET /access-interfaces/:id directly since AccessInterface has its
 * own Detail page.
 */

interface AccessInterfaceDto {
  id: string
  pon_port_id: string
  technology: AccessInterface['technology']
  name: string
  status: AccessInterface['status']
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: AccessInterfaceDto): AccessInterface {
  return {
    id: dto.id,
    ponPortId: dto.pon_port_id,
    technology: dto.technology,
    name: dto.name,
    status: dto.status,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listAccessInterfaces(): Promise<AccessInterface[]> {
  const { access_interfaces: accessInterfaces } = await apiFetch<{ access_interfaces: AccessInterfaceDto[] }>('/access-interfaces/')
  return accessInterfaces.map(fromDto)
}

export async function listAccessInterfacesByPONPortId(ponPortId: string): Promise<AccessInterface[]> {
  const accessInterfaces = await listAccessInterfaces()
  return accessInterfaces.filter((accessInterface) => accessInterface.ponPortId === ponPortId)
}

/** Fetches a single AccessInterface, returning null (not throwing) when it does not exist. */
export async function getAccessInterfaceById(id: string): Promise<AccessInterface | null> {
  try {
    const dto = await apiFetch<AccessInterfaceDto>(`/access-interfaces/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateAccessInterfaceInput {
  ponPortId: string
  technology: AccessInterface['technology']
  name: string
  status: AccessInterface['status']
  description: string
}

export async function createAccessInterface(input: CreateAccessInterfaceInput): Promise<AccessInterface> {
  const dto = await apiFetch<AccessInterfaceDto>('/access-interfaces/', {
    method: 'POST',
    body: {
      pon_port_id: input.ponPortId,
      technology: input.technology,
      name: input.name,
      status: input.status,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the AccessInterface identified by id. access_interfaces.id is
 * referenced by access_attachments.access_interface_id ON DELETE
 * RESTRICT, so this throws an ApiError with kind "conflict" if the
 * Access Interface still has any Access Attachment.
 */
export async function deleteAccessInterface(id: string): Promise<void> {
  await apiFetch<void>(`/access-interfaces/${id}`, { method: 'DELETE' })
}
