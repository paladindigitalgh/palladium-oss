import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listNotes, createNote } from './noteRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listNotes', () => {
  it('builds the query string from the given, URL-encoded entity type and id', async () => {
    apiFetch.mockResolvedValue({ notes: [] })

    await listNotes('service equipment', 's1/special')

    expect(apiFetch).toHaveBeenCalledWith('/notes/?entity_type=service%20equipment&entity_id=s1%2Fspecial')
  })

  it('maps every note from the DTO', async () => {
    apiFetch.mockResolvedValue({
      notes: [
        {
          id: 'n1',
          entity_type: 'customer',
          entity_id: 'c1',
          author_user_id: 'u1',
          author_email: 'jane@example.com',
          body: 'Called the customer back, issue resolved.',
          created_at: '2026-01-02T00:00:00Z',
        },
      ],
    })

    const result = await listNotes('customer', 'c1')

    expect(result).toEqual([
      {
        id: 'n1',
        entityType: 'customer',
        entityId: 'c1',
        authorUserId: 'u1',
        authorEmail: 'jane@example.com',
        body: 'Called the customer back, issue resolved.',
        createdAt: '2026-01-02T00:00:00Z',
      },
    ])
  })
})

describe('createNote', () => {
  it('sends the request body in the API wire shape, without an author field', async () => {
    apiFetch.mockResolvedValue({
      id: 'new',
      entity_type: 'customer',
      entity_id: 'c1',
      author_user_id: 'u1',
      author_email: 'jane@example.com',
      body: 'New note',
      created_at: '2026-01-02T00:00:00Z',
    })

    const result = await createNote({ entityType: 'customer', entityId: 'c1', body: 'New note' })

    expect(apiFetch).toHaveBeenCalledWith('/notes/', {
      method: 'POST',
      body: {
        entity_type: 'customer',
        entity_id: 'c1',
        body: 'New note',
      },
    })
    expect(result.id).toBe('new')
    expect(result.authorEmail).toBe('jane@example.com')
  })
})
