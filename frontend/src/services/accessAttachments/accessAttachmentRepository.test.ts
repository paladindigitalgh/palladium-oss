import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listAccessAttachments,
  listAccessAttachmentsByAccessInterfaceId,
  listAccessAttachmentsByServiceEquipmentId,
  getActiveAccessAttachmentByServiceEquipmentId,
  createAccessAttachment,
  updateAccessAttachment,
} from './accessAttachmentRepository'

/**
 * No get-by-id or delete here -- see this file's own doc comment: an
 * attachment has no Detail page and is detached (a PUT), not deleted.
 * createAccessAttachment stamps installedAt itself, so that test checks
 * the sent value is a real timestamp rather than asserting an exact one.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

function attachmentDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'aa1',
    access_interface_id: 'ai1',
    service_equipment_id: 'se1',
    installed_at: '2026-01-01T00:00:00Z',
    removed_at: null,
    removal_reason: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listAccessAttachments', () => {
  it('maps every access attachment from the DTO', async () => {
    apiFetch.mockResolvedValue({ access_attachments: [attachmentDto({ id: 'aa1' }), attachmentDto({ id: 'aa2' })] })

    const result = await listAccessAttachments()

    expect(result.map((a) => a.id)).toEqual(['aa1', 'aa2'])
  })
})

describe('listAccessAttachmentsByAccessInterfaceId', () => {
  it('returns only attachments belonging to the given access interface', async () => {
    apiFetch.mockResolvedValue({
      access_attachments: [
        attachmentDto({ id: 'aa1', access_interface_id: 'ai1' }),
        attachmentDto({ id: 'aa2', access_interface_id: 'ai2' }),
      ],
    })

    const result = await listAccessAttachmentsByAccessInterfaceId('ai1')

    expect(result.map((a) => a.id)).toEqual(['aa1'])
  })
})

describe('listAccessAttachmentsByServiceEquipmentId', () => {
  it('returns only attachments belonging to the given service equipment', async () => {
    apiFetch.mockResolvedValue({
      access_attachments: [
        attachmentDto({ id: 'aa1', service_equipment_id: 'se1' }),
        attachmentDto({ id: 'aa2', service_equipment_id: 'se2' }),
      ],
    })

    const result = await listAccessAttachmentsByServiceEquipmentId('se2')

    expect(result.map((a) => a.id)).toEqual(['aa2'])
  })
})

describe('getActiveAccessAttachmentByServiceEquipmentId', () => {
  it('returns the attachment whose removedAt is null', async () => {
    apiFetch.mockResolvedValue({
      access_attachments: [
        attachmentDto({ id: 'aa1', service_equipment_id: 'se1', removed_at: '2026-02-01T00:00:00Z' }),
        attachmentDto({ id: 'aa2', service_equipment_id: 'se1', removed_at: null }),
      ],
    })

    const result = await getActiveAccessAttachmentByServiceEquipmentId('se1')

    expect(result?.id).toBe('aa2')
  })

  it('returns null when every attachment for the equipment has been removed', async () => {
    apiFetch.mockResolvedValue({
      access_attachments: [attachmentDto({ id: 'aa1', service_equipment_id: 'se1', removed_at: '2026-02-01T00:00:00Z' })],
    })

    const result = await getActiveAccessAttachmentByServiceEquipmentId('se1')

    expect(result).toBeNull()
  })

  it('returns null when the equipment has no attachment at all', async () => {
    apiFetch.mockResolvedValue({ access_attachments: [] })

    const result = await getActiveAccessAttachmentByServiceEquipmentId('se1')

    expect(result).toBeNull()
  })
})

describe('createAccessAttachment', () => {
  it('sends the request body in the API wire shape, stamping installedAt and leaving removedAt/removalReason empty', async () => {
    apiFetch.mockResolvedValue(attachmentDto({ id: 'new' }))

    await createAccessAttachment({ accessInterfaceId: 'ai1', serviceEquipmentId: 'se1' })

    expect(apiFetch).toHaveBeenCalledWith('/access-attachments/', {
      method: 'POST',
      body: {
        access_interface_id: 'ai1',
        service_equipment_id: 'se1',
        installed_at: expect.any(String),
        removed_at: null,
        removal_reason: '',
      },
    })
  })
})

describe('updateAccessAttachment', () => {
  it('sends every field as a full-row PUT, including the passed-through unchanged fields', async () => {
    apiFetch.mockResolvedValue(attachmentDto({ id: 'aa1', removed_at: '2026-03-01T00:00:00Z', removal_reason: 'ONT swapped' }))

    await updateAccessAttachment('aa1', {
      accessInterfaceId: 'ai1',
      serviceEquipmentId: 'se1',
      installedAt: '2026-01-01T00:00:00Z',
      removedAt: '2026-03-01T00:00:00Z',
      removalReason: 'ONT swapped',
    })

    expect(apiFetch).toHaveBeenCalledWith('/access-attachments/aa1', {
      method: 'PUT',
      body: {
        access_interface_id: 'ai1',
        service_equipment_id: 'se1',
        installed_at: '2026-01-01T00:00:00Z',
        removed_at: '2026-03-01T00:00:00Z',
        removal_reason: 'ONT swapped',
      },
    })
  })
})
