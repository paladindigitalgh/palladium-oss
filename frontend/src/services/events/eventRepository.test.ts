import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listEvents } from './eventRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listEvents', () => {
  it('builds the query string from the given, URL-encoded entity type and id', async () => {
    apiFetch.mockResolvedValue({ events: [] })

    await listEvents('service equipment', 's1/special')

    expect(apiFetch).toHaveBeenCalledWith('/events/?entity_type=service%20equipment&entity_id=s1%2Fspecial')
  })

  it('maps the DTO, passing metadata through as-is including null', async () => {
    apiFetch.mockResolvedValue({
      events: [
        {
          id: 'e1',
          entity_type: 'service',
          entity_id: 's1',
          type: 'service.activated',
          message: 'Service activated',
          metadata: { workflowId: 'w1' },
          actor_user_id: 'u1',
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 'e2',
          entity_type: 'service',
          entity_id: 's1',
          type: 'service.created',
          message: 'Service created',
          metadata: null,
          actor_user_id: null,
          created_at: '2026-01-01T00:00:00Z',
        },
      ],
    })

    const result = await listEvents('service', 's1')

    expect(result[0]).toEqual({
      id: 'e1',
      entityType: 'service',
      entityId: 's1',
      type: 'service.activated',
      message: 'Service activated',
      metadata: { workflowId: 'w1' },
      actorUserId: 'u1',
      createdAt: '2026-01-01T00:00:00Z',
    })
    expect(result[1].metadata).toBeNull()
    expect(result[1].actorUserId).toBeNull()
  })
})
