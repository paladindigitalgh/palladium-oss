import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  listUsers,
  createUser,
  updateUserRole,
  deactivateUser,
  reactivateUser,
  getCurrentUser,
  updateCurrentUserName,
  changeCurrentUserPassword,
} from './userRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listUsers', () => {
  it('maps every user from the DTO', async () => {
    apiFetch.mockResolvedValue({
      users: [
        {
          id: 'u1',
          email: 'admin@example.com',
          first_name: 'Ada',
          last_name: 'Min',
          role: 'Administrator',
          status: 'Active',
          created_at: '2026-01-01T00:00:00Z',
        },
        {
          id: 'u2',
          email: 'viewer@example.com',
          first_name: '',
          last_name: '',
          role: 'Viewer',
          status: 'Inactive',
          created_at: '2026-01-02T00:00:00Z',
        },
      ],
    })

    const result = await listUsers()

    expect(apiFetch).toHaveBeenCalledWith('/users/')
    expect(result).toEqual([
      {
        id: 'u1',
        email: 'admin@example.com',
        firstName: 'Ada',
        lastName: 'Min',
        role: 'Administrator',
        status: 'Active',
        createdAt: '2026-01-01T00:00:00Z',
      },
      {
        id: 'u2',
        email: 'viewer@example.com',
        firstName: '',
        lastName: '',
        role: 'Viewer',
        status: 'Inactive',
        createdAt: '2026-01-02T00:00:00Z',
      },
    ])
  })
})

describe('createUser', () => {
  it('sends the request body in the API wire shape, with no status field', async () => {
    apiFetch.mockResolvedValue({
      id: 'new',
      email: 'jane@example.com',
      first_name: 'Jane',
      last_name: 'Doe',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await createUser({
      email: 'jane@example.com',
      password: 'correct horse battery staple',
      firstName: 'Jane',
      lastName: 'Doe',
      role: 'Operator',
    })

    expect(apiFetch).toHaveBeenCalledWith('/users/', {
      method: 'POST',
      body: {
        email: 'jane@example.com',
        password: 'correct horse battery staple',
        first_name: 'Jane',
        last_name: 'Doe',
        role: 'Operator',
      },
    })
    expect(result.id).toBe('new')
    expect(result.firstName).toBe('Jane')
    expect(result.lastName).toBe('Doe')
  })

  it('defaults first/last name to empty strings when omitted', async () => {
    apiFetch.mockResolvedValue({
      id: 'new',
      email: 'jane@example.com',
      first_name: '',
      last_name: '',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    await createUser({ email: 'jane@example.com', password: 'correct horse battery staple', role: 'Operator' })

    expect(apiFetch).toHaveBeenCalledWith('/users/', {
      method: 'POST',
      body: { email: 'jane@example.com', password: 'correct horse battery staple', first_name: '', last_name: '', role: 'Operator' },
    })
  })
})

describe('updateUserRole', () => {
  it('PUTs the new role', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      first_name: '',
      last_name: '',
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
      first_name: '',
      last_name: '',
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
      first_name: '',
      last_name: '',
      role: 'Viewer',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await reactivateUser('u1')

    expect(apiFetch).toHaveBeenCalledWith('/users/u1/reactivate', { method: 'POST' })
    expect(result.status).toBe('Active')
  })
})

describe('getCurrentUser', () => {
  it('GETs /me and maps the DTO', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      first_name: 'Jane',
      last_name: 'Doe',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await getCurrentUser()

    expect(apiFetch).toHaveBeenCalledWith('/me/')
    expect(result.firstName).toBe('Jane')
    expect(result.lastName).toBe('Doe')
  })
})

describe('updateCurrentUserName', () => {
  it('PUTs the new name to /me', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      first_name: 'Jane',
      last_name: 'Doe',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    const result = await updateCurrentUserName({ firstName: 'Jane', lastName: 'Doe' })

    expect(apiFetch).toHaveBeenCalledWith('/me/', {
      method: 'PUT',
      body: { first_name: 'Jane', last_name: 'Doe' },
    })
    expect(result.firstName).toBe('Jane')
  })
})

describe('changeCurrentUserPassword', () => {
  it('PUTs current/new password to /me/password', async () => {
    apiFetch.mockResolvedValue({
      id: 'u1',
      email: 'jane@example.com',
      first_name: '',
      last_name: '',
      role: 'Operator',
      status: 'Active',
      created_at: '2026-01-01T00:00:00Z',
    })

    await changeCurrentUserPassword({ currentPassword: 'old', newPassword: 'new' })

    expect(apiFetch).toHaveBeenCalledWith('/me/password', {
      method: 'PUT',
      body: { current_password: 'old', new_password: 'new' },
    })
  })
})
