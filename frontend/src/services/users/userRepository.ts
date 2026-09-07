import type { User } from '@/types/user'
import { apiFetch } from '@/services/api/httpClient'

interface UserDto {
  id: string
  email: string
  role: User['role']
  status: User['status']
  created_at: string
}

function fromDto(dto: UserDto): User {
  return { id: dto.id, email: dto.email, role: dto.role, status: dto.status, createdAt: dto.created_at }
}

export async function listUsers(): Promise<User[]> {
  const { users } = await apiFetch<{ users: UserDto[] }>('/users/')
  return users.map(fromDto)
}

export interface CreateUserInput {
  email: string
  password: string
  role: User['role']
}

/**
 * Creates a User. There is no Status field in the input: a freshly
 * created account always starts 'Active' (see
 * internal/auth/service.UserManagementService.Create) -- this form has
 * no way to create a pre-deactivated one.
 */
export async function createUser(input: CreateUserInput): Promise<User> {
  const dto = await apiFetch<UserDto>('/users/', {
    method: 'POST',
    body: { email: input.email, password: input.password, role: input.role },
  })
  return fromDto(dto)
}

export async function updateUserRole(id: string, role: User['role']): Promise<User> {
  const dto = await apiFetch<UserDto>(`/users/${id}/role`, {
    method: 'PUT',
    body: { role },
  })
  return fromDto(dto)
}

/**
 * Deactivates a User: they can no longer log in, and lose access to
 * every capability-gated endpoint on their very next request (see
 * authz.Middleware.Require) -- but the record itself is never deleted.
 */
export async function deactivateUser(id: string): Promise<User> {
  const dto = await apiFetch<UserDto>(`/users/${id}/deactivate`, { method: 'POST' })
  return fromDto(dto)
}

export async function reactivateUser(id: string): Promise<User> {
  const dto = await apiFetch<UserDto>(`/users/${id}/reactivate`, { method: 'POST' })
  return fromDto(dto)
}
