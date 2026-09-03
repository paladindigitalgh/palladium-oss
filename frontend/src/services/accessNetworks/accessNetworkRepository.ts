import type { AccessNetwork } from '@/types/accessNetwork'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real AccessNetwork data source, top of the access-network
 * hierarchy -- mirrors customerRepository.ts's shape exactly. GET
 * /access-networks has no server-side filtering (see
 * internal/accessnetwork/httpapi), so search/sort/pagination happen
 * client-side.
 */

interface AccessNetworkDto {
  id: string
  name: string
  status: AccessNetwork['status']
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: AccessNetworkDto): AccessNetwork {
  return {
    id: dto.id,
    name: dto.name,
    status: dto.status,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export interface AccessNetworkListQuery {
  search?: string
  status?: AccessNetwork['status'] | 'all'
  sortKey?: 'name' | 'status'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface AccessNetworkListResult {
  items: AccessNetwork[]
  total: number
}

function matchesSearch(accessNetwork: AccessNetwork, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return accessNetwork.name.toLowerCase().includes(needle) || accessNetwork.id.toLowerCase().includes(needle)
}

function compareAccessNetworks(sortKey: NonNullable<AccessNetworkListQuery['sortKey']>, direction: number) {
  return (a: AccessNetwork, b: AccessNetwork): number => {
    const comparison = sortKey === 'status' ? a.status.localeCompare(b.status) : a.name.localeCompare(b.name)
    return comparison * direction
  }
}

/** Fetches every AccessNetwork and applies search/filter/sort/pagination client-side. */
export async function listAccessNetworks(query: AccessNetworkListQuery = {}): Promise<AccessNetworkListResult> {
  const { search = '', status = 'all', sortKey = 'name', sortDirection = 'asc', page = 1, pageSize = 15 } = query

  const { access_networks: accessNetworks } = await apiFetch<{ access_networks: AccessNetworkDto[] }>('/access-networks/')
  let results = accessNetworks.map(fromDto).filter((accessNetwork) => matchesSearch(accessNetwork, search))

  if (status !== 'all') results = results.filter((accessNetwork) => accessNetwork.status === status)

  results = results.slice().sort(compareAccessNetworks(sortKey, sortDirection === 'desc' ? -1 : 1))

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

/** Fetches a single AccessNetwork, returning null (not throwing) when it does not exist. */
export async function getAccessNetworkById(id: string): Promise<AccessNetwork | null> {
  try {
    const dto = await apiFetch<AccessNetworkDto>(`/access-networks/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateAccessNetworkInput {
  name: string
  status: AccessNetwork['status']
  description: string
}

export async function createAccessNetwork(input: CreateAccessNetworkInput): Promise<AccessNetwork> {
  const dto = await apiFetch<AccessNetworkDto>('/access-networks/', {
    method: 'POST',
    body: { name: input.name, status: input.status, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the AccessNetwork identified by id. access_networks.id is
 * referenced by olts.access_network_id ON DELETE RESTRICT, so this throws
 * an ApiError with kind "conflict" if the access network still has any
 * OLT -- callers should catch that and explain it, not treat it as an
 * unexpected failure.
 */
export async function deleteAccessNetwork(id: string): Promise<void> {
  await apiFetch<void>(`/access-networks/${id}`, { method: 'DELETE' })
}
