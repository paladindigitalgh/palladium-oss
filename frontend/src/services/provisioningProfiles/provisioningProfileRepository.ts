import type { ProvisioningProfile } from '@/types/provisioningProfile'
import { apiFetch } from '@/services/api/httpClient'

/**
 * GET /provisioning-profiles has no server-side filtering, so every list
 * below fetches the full set once and filters client-side -- the same
 * pattern serviceEquipmentRepository.ts uses, and appropriate here for
 * the same reason: this domain's expected size is one row per Product
 * per vendor, not a large table.
 */
interface ProvisioningProfileDto {
  id: string
  product_id: string
  vendor: string
  profile_name: string
  description: string
}

function fromDto(dto: ProvisioningProfileDto): ProvisioningProfile {
  return { id: dto.id, productId: dto.product_id, vendor: dto.vendor, profileName: dto.profile_name, description: dto.description }
}

export async function listProvisioningProfiles(): Promise<ProvisioningProfile[]> {
  const { provisioning_profiles: profiles } = await apiFetch<{ provisioning_profiles: ProvisioningProfileDto[] }>(
    '/provisioning-profiles/',
  )
  return profiles.map(fromDto)
}

export interface CreateProvisioningProfileInput {
  productId: string
  vendor: string
  profileName: string
  description: string
}

/**
 * Maps productId to the OLT profile name an operator already configured
 * by hand (see this domain's own package doc comment) -- Palladium never
 * generates or pushes this profile to the OLT itself.
 */
export async function createProvisioningProfile(input: CreateProvisioningProfileInput): Promise<ProvisioningProfile> {
  const dto = await apiFetch<ProvisioningProfileDto>('/provisioning-profiles/', {
    method: 'POST',
    body: {
      product_id: input.productId,
      vendor: input.vendor,
      profile_name: input.profileName,
      description: input.description,
    },
  })
  return fromDto(dto)
}
