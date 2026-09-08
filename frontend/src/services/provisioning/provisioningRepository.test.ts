import { describe, it, expect, vi, beforeEach } from 'vitest'
import { authorizeONU, deauthorizeONU } from './provisioningRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

beforeEach(() => {
  apiFetch.mockReset()
})

describe('authorizeONU', () => {
  it('posts port and serial_number to the authorize-onu endpoint and returns the interface', async () => {
    apiFetch.mockResolvedValue({ interface: 'xgs/6/3' })

    const result = await authorizeONU('olt1', 'xgs/6', 'ISKT2308DD88')

    expect(apiFetch).toHaveBeenCalledWith('/provisioning/olts/olt1/authorize-onu', {
      method: 'POST',
      body: { port: 'xgs/6', serial_number: 'ISKT2308DD88' },
    })
    expect(result).toBe('xgs/6/3')
  })
})

describe('deauthorizeONU', () => {
  it('posts to the deauthorize-onu endpoint and returns the interface', async () => {
    apiFetch.mockResolvedValue({ interface: 'xgs/6/3' })

    const result = await deauthorizeONU('device1')

    expect(apiFetch).toHaveBeenCalledWith('/provisioning/devices/device1/deauthorize-onu', { method: 'POST' })
    expect(result).toBe('xgs/6/3')
  })
})
