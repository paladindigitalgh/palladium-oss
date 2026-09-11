<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseCheckbox from '@/components/base/BaseCheckbox.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import UserFormDialog from '@/components/dialogs/UserFormDialog.vue'
import { listUsers, updateUserRole, deactivateUser, reactivateUser } from '@/services/users/userRepository'
import { ApiError } from '@/services/api/httpClient'
import { formatDisplayName } from '@/lib/users'
import type { User } from '@/types/user'

/**
 * The Users page, reached from the Administration landing hub
 * (AdministrationView.vue). Manages platform login identities
 * (internal/auth) -- independent of Providers/Plans, which have nothing
 * to do with who can sign in. Split out of what was previously a single
 * stacked-panel Administration page into its own route
 * (/administration/users) at the user's explicit request; see
 * AdministrationView.vue's own doc comment for why this deviates from
 * docs/09-WORKSPACE-SPECIFICATIONS.md section 16's original "flat
 * Primary Panels, one page" description of Administration.
 *
 * There is no Delete -- both events and workflow instances can
 * reference a User, so Deactivate/Reactivate (internal/auth's
 * UserStatus) stands in for it, the same soft-removal reasoning
 * internal/serviceequipment already uses (see
 * docs/03-DOMAIN-MODEL.md section 7). Role is edited inline via a
 * BaseSelect per row rather than a dialog: there is no other
 * inline-edit-in-a-table pattern yet in this codebase to reuse, and a
 * BaseSelect control is the simplest option for a single-field change.
 */
const users = ref<User[]>([])
const loading = ref(true)

onMounted(async () => {
  loading.value = true
  users.value = await listUsers()
  loading.value = false
})

/**
 * Inactive accounts are excluded from the default view -- the common
 * workflow only cares about who can currently sign in -- with a
 * checkbox to bring them back for the rarer "review who's been
 * deactivated" task.
 */
const includeInactive = ref(false)
const visibleUsers = computed(() =>
  includeInactive.value ? users.value : users.value.filter((user) => user.status !== 'Inactive'),
)

const userColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Name' },
  { key: 'email', label: 'Email' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

const roleOptions = [
  { value: 'Viewer', label: 'Viewer' },
  { value: 'Operator', label: 'Operator' },
  { value: 'Administrator', label: 'Administrator' },
]

const showUserForm = ref(false)
const userActionError = ref<string | null>(null)

function handleUserCreated(user: User) {
  showUserForm.value = false
  users.value = [...users.value, user]
}

async function handleRoleChange(user: User, role: string) {
  userActionError.value = null
  try {
    const updated = await updateUserRole(user.id, role as User['role'])
    users.value = users.value.map((u) => (u.id === updated.id ? updated : u))
  } catch (err) {
    userActionError.value = err instanceof ApiError ? err.message : 'The role could not be changed.'
  }
}

async function handleToggleStatus(user: User) {
  userActionError.value = null
  try {
    const updated = user.status === 'Active' ? await deactivateUser(user.id) : await reactivateUser(user.id)
    users.value = users.value.map((u) => (u.id === updated.id ? updated : u))
  } catch (err) {
    userActionError.value = err instanceof ApiError ? err.message : 'The account status could not be changed.'
  }
}
</script>

<template>
  <div class="administration-users-view">
    <WorkspaceHeader title="Users" :breadcrumbs="[{ label: 'Administration', to: '/administration' }]">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showUserForm = true">New User</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <p class="page-description">
      Platform login accounts. There is no delete -- deactivate an account to block login and remove access without
      losing its history.
    </p>

    <UserFormDialog :open="showUserForm" @close="showUserForm = false" @created="handleUserCreated" />

    <p v-if="userActionError" class="user-action-error" role="alert">{{ userActionError }}</p>

    <BaseCheckbox v-model="includeInactive" label="Include Inactive" />

    <BaseCard>
      <div v-if="loading" class="page-status">
        <BaseLoadingState :lines="3" />
      </div>

      <SimpleTable
        v-else
        :columns="userColumns"
        :rows="visibleUsers"
        :row-key="(user) => user.id"
        empty-icon="settings"
        empty-title="No users yet"
      >
        <template #cell-name="{ row }">
          {{ formatDisplayName({ firstName: row.firstName, lastName: row.lastName, email: row.email }) }}
        </template>
        <template #cell-email="{ row }">{{ row.email }}</template>
        <template #cell-role="{ row }">
          <BaseSelect
            :model-value="row.role"
            label="Role"
            hide-label
            :options="roleOptions"
            @update:model-value="(value) => handleRoleChange(row, value)"
          />
        </template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="secondary" size="sm" @click="handleToggleStatus(row)">
            {{ row.status === 'Active' ? 'Deactivate' : 'Reactivate' }}
          </BaseButton>
        </template>
      </SimpleTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.administration-users-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-description {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.page-status {
  padding: var(--space-4) 0;
}

.user-action-error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
