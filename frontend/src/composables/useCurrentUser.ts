import { ref } from 'vue'
import type { User } from '@/types/user'

/**
 * The signed-in caller's own full User record (id, email, FirstName/
 * LastName, role, status) -- App-owned state, the same module-scope-
 * singleton pattern useAuth.ts's token/claims and useTheme.ts's theme
 * already use, so every component reads the same instance rather than
 * each fetching its own copy.
 *
 * This is deliberately separate from useAuth.ts: that composable is
 * scoped to the session itself (the JWT, decoded claims, login/logout),
 * and a JWT carries only ID and Email (see auth.Claims's doc comment) --
 * never FirstName/LastName, which can change mid-session (see the
 * Profile screen, ProfileEditDialog.vue). This composable is what
 * UserMenu.vue and ProfileEditDialog.vue both read for "who am I,
 * currently" beyond what the token alone can answer.
 */
const user = ref<User | null>(null)

export function useCurrentUser() {
  /** Fetches GET /api/v1/me and stores the result. */
  async function refresh(): Promise<User> {
    const { getCurrentUser } = await import('@/services/users/userRepository')
    const fetched = await getCurrentUser()
    user.value = fetched
    return fetched
  }

  /** Sets the cached record directly -- called after UpdateName/ChangePassword's response, so every reader updates immediately without a second round trip. */
  function set(next: User): void {
    user.value = next
  }

  /** Called on sign-out so a stale name never survives into the next session. */
  function clear(): void {
    user.value = null
  }

  return { user, refresh, set, clear }
}
