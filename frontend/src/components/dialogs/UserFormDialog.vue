<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createUser } from '@/services/users/userRepository'
import { ApiError } from '@/services/api/httpClient'
import type { User } from '@/types/user'

/**
 * Creates a User account. Create-only, like ProviderFormDialog.vue:
 * there is no "edit a user's email or password" flow here -- Role is
 * changed inline in the Users table on AdministrationView.vue,
 * deactivating/reactivating are separate actions there too, and a User
 * changes their own name/password themselves via the Profile screen
 * (ProfileEditDialog.vue, reached from UserMenu.vue), not through this
 * Administrator-only form.
 *
 * There is no Status field: a freshly created account always starts
 * Active (see internal/auth/service.UserManagementService.Create), and
 * the initial password is typed directly into this form by the
 * Administrator creating the account -- there is no invite/email
 * infrastructure to send it through instead, so it must be relayed to
 * the new user out of band. First/Last Name are both optional (see
 * internal/auth.User's doc comment) -- an Administrator can leave either
 * or both blank, and the new User can fill them in later themselves.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', user: User): void
}>()

const roleOptions = [
  { value: 'Viewer', label: 'Viewer' },
  { value: 'Operator', label: 'Operator' },
  { value: 'Administrator', label: 'Administrator' },
]

const email = ref('')
const password = ref('')
const firstName = ref('')
const lastName = ref('')
const role = ref<User['role']>('Viewer')
const submitting = ref(false)
const error = ref<string | null>(null)

function reset() {
  email.value = ''
  password.value = ''
  firstName.value = ''
  lastName.value = ''
  role.value = 'Viewer'
  error.value = null
}

watch(
  () => props.open,
  (open) => {
    if (open) reset()
  },
  { immediate: true },
)

function close() {
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
    const user = await createUser({
      email: email.value,
      password: password.value,
      firstName: firstName.value,
      lastName: lastName.value,
      role: role.value,
    })
    reset()
    emit('created', user)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The user could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New User" @close="close">
    <form class="user-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="email" type="email" label="Email" placeholder="jane@example.com" required />
      <BaseInput v-model="password" type="password" label="Initial Password" required />
      <BaseInput v-model="firstName" label="First Name" placeholder="Optional" />
      <BaseInput v-model="lastName" label="Last Name" placeholder="Optional" />
      <BaseSelect v-model="role" label="Role" :options="roleOptions" />

      <p v-if="error" class="user-form__error" role="alert">{{ error }}</p>

      <div class="user-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !email || !password">
          {{ submitting ? 'Creating…' : 'Create User' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.user-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.user-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.user-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
