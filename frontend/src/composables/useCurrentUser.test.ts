import { it, expect, vi, beforeEach } from 'vitest'
import { useCurrentUser } from './useCurrentUser'
import type { User } from '@/types/user'

const { getCurrentUser } = vi.hoisted(() => ({ getCurrentUser: vi.fn() }))

vi.mock('@/services/users/userRepository', () => ({ getCurrentUser }))

function testUser(overrides: Partial<User> = {}): User {
  return {
    id: 'u1',
    email: 'jane@example.com',
    firstName: 'Jane',
    lastName: 'Doe',
    role: 'Operator',
    status: 'Active',
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  getCurrentUser.mockReset()
  useCurrentUser().clear()
})

it('starts with no cached user', () => {
  expect(useCurrentUser().user.value).toBeNull()
})

it('refresh() fetches GET /me and caches the result', async () => {
  getCurrentUser.mockResolvedValue(testUser())

  const result = await useCurrentUser().refresh()

  expect(result).toEqual(testUser())
  expect(useCurrentUser().user.value).toEqual(testUser())
})

it('set() updates the cache without a network call, visible to every caller of the composable', () => {
  const a = useCurrentUser()
  const b = useCurrentUser()

  a.set(testUser({ firstName: 'Janet' }))

  expect(b.user.value?.firstName).toBe('Janet')
  expect(getCurrentUser).not.toHaveBeenCalled()
})

it('clear() removes the cached user', () => {
  useCurrentUser().set(testUser())

  useCurrentUser().clear()

  expect(useCurrentUser().user.value).toBeNull()
})
