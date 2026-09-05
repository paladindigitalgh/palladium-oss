import type { Provider } from '@/types/provider'
import { apiFetch } from '@/services/api/httpClient'

interface ProviderDto {
  id: string
  name: string
  status: Provider['status']
  description: string
}

function fromDto(dto: ProviderDto): Provider {
  return { id: dto.id, name: dto.name, status: dto.status, description: dto.description }
}

export async function listProviders(): Promise<Provider[]> {
  const { providers } = await apiFetch<{ providers: ProviderDto[] }>('/providers/')
  return providers.map(fromDto)
}

export interface CreateProviderInput {
  name: string
  description: string
}

/**
 * Creates a Provider. Status is always sent as "Active": this dialog
 * only creates a fresh, currently-operating ISP identity, the same
 * reasoning createProduct's Status stamping documents -- deactivating
 * one later is a distinct action this form does not do.
 */
export async function createProvider(input: CreateProviderInput): Promise<Provider> {
  const dto = await apiFetch<ProviderDto>('/providers/', {
    method: 'POST',
    body: {
      name: input.name,
      status: 'Active',
      description: input.description,
    },
  })
  return fromDto(dto)
}
