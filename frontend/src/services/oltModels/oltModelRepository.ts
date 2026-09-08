import type { OLTModel } from '@/types/oltModel'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real OLTModel data source. GET /olt-models has no server-side
 * filtering (see internal/oltmodel/httpapi), the same as
 * providerRepository.ts's listProviders.
 */

interface OLTModelDto {
  id: string
  vendor: OLTModel['vendor']
  name: string
  pon_port_count: number
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: OLTModelDto): OLTModel {
  return {
    id: dto.id,
    vendor: dto.vendor,
    name: dto.name,
    ponPortCount: dto.pon_port_count,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listOLTModels(): Promise<OLTModel[]> {
  const { olt_models: oltModels } = await apiFetch<{ olt_models: OLTModelDto[] }>('/olt-models/')
  return oltModels.map(fromDto)
}

/**
 * Fetches a single OLTModel, returning null (not throwing) when it does
 * not exist -- the same pattern as oltRepository.ts's getOLTById. Used
 * to resolve an OLT's own OLTModelID for display (see
 * OLTDetailView.vue), not by the OLT Model picker in OLTFormDialog.vue,
 * which needs the full list.
 */
export async function getOLTModelById(id: string): Promise<OLTModel | null> {
  try {
    const dto = await apiFetch<OLTModelDto>(`/olt-models/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateOLTModelInput {
  vendor: OLTModel['vendor']
  name: string
  ponPortCount: number
  description: string
}

export async function createOLTModel(input: CreateOLTModelInput): Promise<OLTModel> {
  const dto = await apiFetch<OLTModelDto>('/olt-models/', {
    method: 'POST',
    body: {
      vendor: input.vendor,
      name: input.name,
      pon_port_count: input.ponPortCount,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the OLTModel identified by id. olt_models.id is referenced by
 * olts.olt_model_id ON DELETE RESTRICT, so this throws an ApiError with
 * kind "conflict" if any OLT still references it.
 */
export async function deleteOLTModel(id: string): Promise<void> {
  await apiFetch<void>(`/olt-models/${id}`, { method: 'DELETE' })
}
