import type { Site } from '@/types/site'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Site data source, top of the Inventory hierarchy -- mirrors
 * accessNetworkRepository.ts's shape exactly. GET /sites has no
 * server-side filtering (see internal/inventory/httpapi), so search/
 * sort/pagination happen client-side.
 */

interface SiteDto {
  id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: SiteDto): Site {
  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export interface SiteListQuery {
  search?: string
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface SiteListResult {
  items: Site[]
  total: number
}

function matchesSearch(site: Site, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return site.name.toLowerCase().includes(needle) || site.id.toLowerCase().includes(needle)
}

/** Fetches every Site and applies search/sort/pagination client-side. No status field exists to filter or sort by -- unlike AccessNetwork, Site is Name/Description only. */
export async function listSites(query: SiteListQuery = {}): Promise<SiteListResult> {
  const { search = '', sortDirection = 'asc', page = 1, pageSize = 15 } = query

  const { sites } = await apiFetch<{ sites: SiteDto[] }>('/sites/')
  let results = sites.map(fromDto).filter((site) => matchesSearch(site, search))

  const direction = sortDirection === 'desc' ? -1 : 1
  results = results.slice().sort((a, b) => a.name.localeCompare(b.name) * direction)

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

/** Fetches a single Site, returning null (not throwing) when it does not exist. */
export async function getSiteById(id: string): Promise<Site | null> {
  try {
    const dto = await apiFetch<SiteDto>(`/sites/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateSiteInput {
  name: string
  description: string
}

export async function createSite(input: CreateSiteInput): Promise<Site> {
  const dto = await apiFetch<SiteDto>('/sites/', {
    method: 'POST',
    body: { name: input.name, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdateSiteInput {
  name: string
  description: string
}

export async function updateSite(id: string, input: UpdateSiteInput): Promise<Site> {
  const dto = await apiFetch<SiteDto>(`/sites/${id}`, {
    method: 'PUT',
    body: { name: input.name, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the Site identified by id. sites.id is referenced by
 * buildings.site_id ON DELETE RESTRICT, so this throws an ApiError with
 * kind "conflict" if the site still has any Building -- callers should
 * catch that and explain it, not treat it as an unexpected failure.
 */
export async function deleteSite(id: string): Promise<void> {
  await apiFetch<void>(`/sites/${id}`, { method: 'DELETE' })
}
