import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listProvisioningProfiles, createProvisioningProfile } from './provisioningProfileRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

function profileDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'pp1',
    product_id: 'p1',
    vendor: 'Kontron',
    profile_name: 'RES-500M',
    description: '',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listProvisioningProfiles', () => {
  it('maps every provisioning profile from the DTO', async () => {
    apiFetch.mockResolvedValue({
      provisioning_profiles: [profileDto({ id: 'pp1' }), profileDto({ id: 'pp2', profile_name: 'RES-1000M' })],
    })

    const result = await listProvisioningProfiles()

    expect(apiFetch).toHaveBeenCalledWith('/provisioning-profiles/')
    expect(result).toEqual([
      { id: 'pp1', productId: 'p1', vendor: 'Kontron', profileName: 'RES-500M', description: '' },
      { id: 'pp2', productId: 'p1', vendor: 'Kontron', profileName: 'RES-1000M', description: '' },
    ])
  })
})

describe('createProvisioningProfile', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(profileDto({ id: 'new' }))

    await createProvisioningProfile({
      productId: 'p1',
      vendor: 'Kontron',
      profileName: 'RES-500M',
      description: '500 Mbps residential',
    })

    expect(apiFetch).toHaveBeenCalledWith('/provisioning-profiles/', {
      method: 'POST',
      body: {
        product_id: 'p1',
        vendor: 'Kontron',
        profile_name: 'RES-500M',
        description: '500 Mbps residential',
      },
    })
  })
})
