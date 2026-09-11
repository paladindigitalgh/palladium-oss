import type { User } from '@/types/user'
import { apiFetch } from '@/services/api/httpClient'

interface UserDto {
  id: string
  email: string
  first_name: string
  last_name: string
  role: User['role']
  status: User['status']
  created_at: string
}

function fromDto(dto: UserDto): User {
  return {
    id: dto.id,
    email: dto.email,
    firstName: dto.first_name,
    lastName: dto.last_name,
    role: dto.role,
    status: dto.status,
    createdAt: dto.created_at,
  }
}

export async function listUsers(): Promise<User[]> {
  const { users } = await apiFetch<{ users: UserDto[] }>('/users/')
  return users.map(fromDto)
}

export interface CreateUserInput {
  email: string
  password: string
  firstName?: string
  lastName?: string
  role: User['role']
}

/**
 * Creates a User. There is no Status field in the input: a freshly
 * created account always starts 'Active' (see
 * internal/auth/service.UserManagementService.Create) -- this form has
 * no way to create a pre-deactivated one. firstName/lastName are both
 * optional, matching internal/auth.User's own optional fields.
 */
export async function createUser(input: CreateUserInput): Promise<User> {
  const dto = await apiFetch<UserDto>('/users/', {
    method: 'POST',
    body: {
      email: input.email,
      password: input.password,
      first_name: input.firstName ?? '',
      last_name: input.lastName ?? '',
      role: input.role,
    },
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

/**
 * Fetches the signed-in caller's own account (GET /api/v1/me) -- always
 * whichever User the caller's own JWT names, never a User given by ID
 * (see internal/auth/httpapi.ProfileHandler's doc comment). Backs the
 * Profile screen (ProfileEditDialog.vue) and UserMenu.vue's display
 * name, neither of which the JWT alone can supply: a token carries only
 * ID and email (see auth.Claims), never FirstName/LastName.
 */
export async function getCurrentUser(): Promise<User> {
  const dto = await apiFetch<UserDto>('/me/')
  return fromDto(dto)
}

export interface UpdateCurrentUserNameInput {
  firstName: string
  lastName: string
}

/** Changes the signed-in caller's own FirstName/LastName. */
export async function updateCurrentUserName(input: UpdateCurrentUserNameInput): Promise<User> {
  const dto = await apiFetch<UserDto>('/me/', {
    method: 'PUT',
    body: { first_name: input.firstName, last_name: input.lastName },
  })
  return fromDto(dto)
}

export interface ChangeCurrentUserPasswordInput {
  currentPassword: string
  newPassword: string
}

/**
 * Changes the signed-in caller's own password. Requires currentPassword
 * -- a still-valid session token alone is not enough to change it (see
 * service.ProfileService.ChangePassword's doc comment).
 */
export async function changeCurrentUserPassword(input: ChangeCurrentUserPasswordInput): Promise<User> {
  const dto = await apiFetch<UserDto>('/me/password', {
    method: 'PUT',
    body: { current_password: input.currentPassword, new_password: input.newPassword },
  })
  return fromDto(dto)
}
