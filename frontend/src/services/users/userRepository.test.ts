import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listUsers, createUser, updateUserRole, deactivateUser, reactivateUser } from './userRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listUsers', () => {
  it('maps every user from the DTO', async () => {
    apiFetch.mockResolvedValue({
      users: [
        { id: 'u1', email: 'admin@example.com', role: 'Administrator', status: 'Active', created_at: '2026-01-01T00:00:00Z' },
        { id: 'u2', email: 'viewer@example.com', role: 'Viewer', status: 'Inactive', created_at: '2026-01-02T00:00:00Z' },
      ],
    })

    const result = await listUsers()

    expect(apiFetch).toHaveBeenCalledWith('/users/')
    expect(result).toEqual([
      { id: 'u1', email: 'admin@example.com', role: 'Administrator', status: 'Active', createdAt: '2026-01-01T00:00:00Z' },
      { id: 'u2', email: 'viewer@example.com', role: 'Viewer', status: 'Inactive', createdAt: '2026-01-02T00:00:00Z' },
    ])
  })
})

describe('createUser', () => {
  it('sends the request body in the API wire shape, with no status field', async () => {
    apiFetch.mockResolvedValue({
      id: 'new',
      email: 'jane@example.com',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await createUser({ email: 'jane@example.com', password: 'correct horse battery staple', role: 'Operator' })

    expect(apiFetch).toHaveBeenCalledWith('/users/', {
      method: 'POST',
      body: { email: 'jane@example.com', password: 'correct horse battery staple', role: 'Operator' },
    })
    expect(result.id).toBe('new')
  })
})

describe('updateUserRole', () => {
  it('PUTs the new role', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      role: 'Viewer',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await updateUserRole('u1', 'Viewer')

    expect(apiFetch).toHaveBeenCalledWith('/users/u1/role', { method: 'PUT', body: { role: 'Viewer' } })
    expect(result.role).toBe('Viewer')
  })
})

describe('deactivateUser', () => {
  it('POSTs to the deactivate endpoint', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      role: 'Viewer',
      status: 'Inactive',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await deactivateUser('u1')

    expect(apiFetch).toHaveBeenCalledWith('/users/u1/deactivate', { method: 'POST' })
    expect(result.status).toBe('Inactive')
  })
})

describe('reactivateUser', () => {
  it('POSTs to the reactivate endpoint', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      role: 'Viewer',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await reactivateUser('u1')

    expect(apiFetch).toHaveBeenCalledWith('/users/u1/reactivate', { method: 'POST' })
    expect(result.status).toBe('Active')
  })
})
