import type { Product } from '@/types/product'
import { apiFetch } from '@/services/api/httpClient'

/** Read-only: there is no Product Workspace yet, only the Service creation form's dropdown. */

interface ProductDto {
  id: string
  name: string
  status: Product['status']
}

export async function listProducts(): Promise<Product[]> {
  const { products } = await apiFetch<{ products: ProductDto[] }>('/products/')
  return products.map((dto) => ({ id: dto.id, name: dto.name, status: dto.status }))
}
