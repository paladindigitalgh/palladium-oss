import type { Building } from '@/types/building'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Building data source. GET /buildings has no server-side
 * filtering (see internal/inventory/httpapi), so "a site's Buildings" is
 * resolved by fetching every Building once and filtering client-side --
 * the same pattern oltRepository.ts uses. Building has its own Detail
 * page, so getBuildingById hits GET /buildings/:id directly.
 */

interface BuildingDto {
  id: string
  site_id: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: BuildingDto): Building {
  return {
    id: dto.id,
    siteId: dto.site_id,
    name: dto.name,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listBuildings(): Promise<Building[]> {
  const { buildings } = await apiFetch<{ buildings: BuildingDto[] }>('/buildings/')
  return buildings.map(fromDto)
}

export async function listBuildingsBySiteId(siteId: string): Promise<Building[]> {
  const buildings = await listBuildings()
  return buildings.filter((building) => building.siteId === siteId)
}

/** Fetches a single Building, returning null (not throwing) when it does not exist. */
export async function getBuildingById(id: string): Promise<Building | null> {
  try {
    const dto = await apiFetch<BuildingDto>(`/buildings/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateBuildingInput {
  siteId: string
  name: string
  description: string
}

export async function createBuilding(input: CreateBuildingInput): Promise<Building> {
  const dto = await apiFetch<BuildingDto>('/buildings/', {
    method: 'POST',
    body: { site_id: input.siteId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdateBuildingInput {
  name: string
  description: string
  /**
   * Not user-editable (see BuildingFormDialog.vue -- no site picker
   * exists in this workspace). Callers pass the Building's current
   * siteId through unchanged: PUT replaces every mutable column (see
   * internal/inventory/postgres's Update), so omitting it here would
   * silently move the Building to no site.
   */
  siteId: string
}

export async function updateBuilding(id: string, input: UpdateBuildingInput): Promise<Building> {
  const dto = await apiFetch<BuildingDto>(`/buildings/${id}`, {
    method: 'PUT',
    body: { site_id: input.siteId, name: input.name, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the Building identified by id. buildings.id is referenced by
 * rooms.building_id ON DELETE RESTRICT, so this throws an ApiError with
 * kind "conflict" if the building still has any Room.
 */
export async function deleteBuilding(id: string): Promise<void> {
  await apiFetch<void>(`/buildings/${id}`, { method: 'DELETE' })
}
