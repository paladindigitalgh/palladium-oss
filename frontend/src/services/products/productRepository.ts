import type { Product, ProductCategory } from '@/types/product'
import { apiFetch } from '@/services/api/httpClient'

/**
 * GET /products has no server-side filtering, so listProducts fetches
 * the full set once -- the same pattern serviceEquipmentRepository.ts
 * documents for itself. Two callers today: the Service creation form's
 * dropdown (only ever needed id/name/status), and the Administration
 * Workspace's Plans panel (needs every field to list and edit) -- see
 * types/product.ts's own doc comment on why this type is not trimmed to
 * the smaller of the two anymore.
 */
interface ProductDto {
  id: string
  catalog_id: string
  name: string
  category: ProductCategory
  status: Product['status']
  description: string
}

function fromDto(dto: ProductDto): Product {
  return {
    id: dto.id,
    catalogId: dto.catalog_id,
    name: dto.name,
    category: dto.category,
    status: dto.status,
    description: dto.description,
  }
}

export async function listProducts(): Promise<Product[]> {
  const { products } = await apiFetch<{ products: ProductDto[] }>('/products/')
  return products.map(fromDto)
}

export interface CreateProductInput {
  catalogId: string
  name: string
  category: ProductCategory
  description: string
}

/**
 * Creates a Product -- what the ISP sells (docs/03-DOMAIN-MODEL.md
 * section 5), independent of any vendor's provisioning details. Status
 * is always sent as "Active": this dialog only creates a fresh,
 * currently-sellable offering, the same reasoning
 * createServiceEquipment's installedAt/removedAt stamping documents --
 * retiring a Product later is a distinct action this form does not do.
 */
export async function createProduct(input: CreateProductInput): Promise<Product> {
  const dto = await apiFetch<ProductDto>('/products/', {
    method: 'POST',
    body: {
      catalog_id: input.catalogId,
      name: input.name,
      category: input.category,
      status: 'Active',
      description: input.description,
    },
  })
  return fromDto(dto)
}
