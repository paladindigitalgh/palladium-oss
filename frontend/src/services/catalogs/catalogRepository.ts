import type { Catalog } from '@/types/catalog'
import { apiFetch } from '@/services/api/httpClient'

/** Read-only: there is no Catalog Workspace yet, only PlanFormDialog.vue's default-catalog lookup. */

interface CatalogDto {
  id: string
  name: string
}

export async function listCatalogs(): Promise<Catalog[]> {
  const { catalogs } = await apiFetch<{ catalogs: CatalogDto[] }>('/catalogs/')
  return catalogs.map((dto) => ({ id: dto.id, name: dto.name }))
}
