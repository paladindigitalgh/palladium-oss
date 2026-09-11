<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { updateCurrentUserName, changeCurrentUserPassword } from '@/services/users/userRepository'
import { useCurrentUser } from '@/composables/useCurrentUser'
import { ApiError } from '@/services/api/httpClient'

/**
 * The Profile screen, reached from UserMenu.vue's "Profile" item --
 * every signed-in User's own self-service edit, not an Administrator
 * form: it can only ever act on the caller's own account (see
 * internal/auth/httpapi.ProfileHandler's doc comment on why /me never
 * takes an ID), so unlike UserFormDialog.vue there is no email, role, or
 * status here to change.
 *
 * Two independent forms, each with its own submit/error/success state,
 * because they are two independent operations against two different
 * endpoints (PUT /me, PUT /me/password) that can each succeed or fail on
 * their own -- saving a new name should not require also typing a
 * password, and a failed password change should not discard a
 * successfully saved name.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (event: 'close'): void }>()

const { user, set } = useCurrentUser()

const firstName = ref('')
const lastName = ref('')
const nameSubmitting = ref(false)
const nameError = ref<string | null>(null)
const nameSaved = ref(false)

const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordSubmitting = ref(false)
const passwordError = ref<string | null>(null)
const passwordSaved = ref(false)

watch(
  () => props.open,
  (open) => {
    if (!open) return
    firstName.value = user.value?.firstName ?? ''
    lastName.value = user.value?.lastName ?? ''
    nameError.value = null
    nameSaved.value = false
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordError.value = null
    passwordSaved.value = false
  },
  { immediate: true },
)

function close() {
  emit('close')
}

async function handleSaveName() {
  nameError.value = null
  nameSaved.value = false
  nameSubmitting.value = true
  try {
    const updated = await updateCurrentUserName({ firstName: firstName.value, lastName: lastName.value })
    set(updated)
    nameSaved.value = true
  } catch (err) {
    nameError.value = err instanceof ApiError ? err.message : 'The name could not be saved.'
  } finally {
    nameSubmitting.value = false
  }
}

async function handleChangePassword() {
  passwordError.value = null
  passwordSaved.value = false

  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = 'The new password and confirmation do not match.'
    return
  }

  passwordSubmitting.value = true
  try {
    await changeCurrentUserPassword({ currentPassword: currentPassword.value, newPassword: newPassword.value })
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    passwordSaved.value = true
  } catch (err) {
    passwordError.value = err instanceof ApiError ? err.message : 'The password could not be changed.'
  } finally {
    passwordSubmitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Profile" @close="close">
    <div class="profile-edit">
      <section class="profile-edit__section">
        <h3 class="profile-edit__heading">Name</h3>
        <form class="profile-edit__form" @submit.prevent="handleSaveName">
          <BaseInput v-model="firstName" label="First Name" placeholder="Optional" />
          <BaseInput v-model="lastName" label="Last Name" placeholder="Optional" />

          <p v-if="nameError" class="profile-edit__error" role="alert">{{ nameError }}</p>
          <p v-else-if="nameSaved" class="profile-edit__success">Saved.</p>

          <div class="profile-edit__form-actions">
            <BaseButton type="submit" variant="primary" size="sm" :disabled="nameSubmitting">
              {{ nameSubmitting ? 'Saving…' : 'Save Name' }}
            </BaseButton>
          </div>
        </form>
      </section>

      <section class="profile-edit__section">
        <h3 class="profile-edit__heading">Password</h3>
        <form class="profile-edit__form" @submit.prevent="handleChangePassword">
          <BaseInput v-model="currentPassword" type="password" label="Current Password" required />
          <BaseInput v-model="newPassword" type="password" label="New Password" required />
          <BaseInput v-model="confirmPassword" type="password" label="Confirm New Password" required />

          <p v-if="passwordError" class="profile-edit__error" role="alert">{{ passwordError }}</p>
          <p v-else-if="passwordSaved" class="profile-edit__success">Password changed.</p>

          <div class="profile-edit__form-actions">
            <BaseButton
              type="submit"
              variant="primary"
              size="sm"
              :disabled="passwordSubmitting || !currentPassword || !newPassword || !confirmPassword"
            >
              {{ passwordSubmitting ? 'Changing…' : 'Change Password' }}
            </BaseButton>
          </div>
        </form>
      </section>
    </div>
  </BaseModal>
</template>

<style scoped>
.profile-edit {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.profile-edit__section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.profile-edit__heading {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.profile-edit__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.profile-edit__error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.profile-edit__success {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-success);
}

.profile-edit__form-actions {
  display: flex;
  justify-content: flex-end;
}
</style>
