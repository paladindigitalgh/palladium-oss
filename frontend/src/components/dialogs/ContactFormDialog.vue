<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createContact, updateContact } from '@/services/contacts/contactRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Contact } from '@/types/contact'

/**
 * Dual-mode: create when `contact` is absent, edit when present -- built
 * dual-mode from the start (unlike Location/Device/etc., which got edit
 * as a separate follow-up), mirroring DeviceFormDialog.vue/
 * LocationFormDialog.vue's now-proven shape. `customerId` (the parent
 * prop, needed for create since a new Contact has no customer of its own
 * yet) is ignored in edit mode -- the contact being edited already has
 * one, and moving a Contact to a different Customer is not something
 * this dialog does.
 */
const props = defineProps<{ open: boolean; customerId: string; contact?: Contact | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', contact: Contact): void
  (event: 'updated', contact: Contact): void
}>()

const name = ref('')
const role = ref<Contact['role']>('Primary')
const email = ref('')
const phone = ref('')
const status = ref<Contact['status']>('Active')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const roleOptions = [
  { value: 'Primary', label: 'Primary' },
  { value: 'Billing', label: 'Billing' },
  { value: 'Technical', label: 'Technical' },
  { value: 'Emergency', label: 'Emergency' },
  { value: 'Other', label: 'Other' },
]

const statusOptions = [
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
]

function reset() {
  name.value = ''
  role.value = 'Primary'
  email.value = ''
  phone.value = ''
  status.value = 'Active'
  description.value = ''
  error.value = null
}

function populateFrom(contact: Contact) {
  name.value = contact.name
  role.value = contact.role
  email.value = contact.email
  phone.value = contact.phone
  status.value = contact.status
  description.value = contact.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `contact`
// when editing or blank when creating -- not just once on mount, since
// the same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.contact) populateFrom(props.contact)
    else reset()
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
    if (props.contact) {
      const updated = await updateContact(props.contact.id, {
        customerId: props.contact.customerId,
        name: name.value,
        role: role.value,
        email: email.value,
        phone: phone.value,
        status: status.value,
        description: description.value,
      })
      emit('updated', updated)
    } else {
      const contact = await createContact({
        customerId: props.customerId,
        name: name.value,
        role: role.value,
        email: email.value,
        phone: phone.value,
        status: status.value,
        description: description.value,
      })
      reset()
      emit('created', contact)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The contact could not be ${props.contact ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="contact ? 'Edit Contact' : 'Add Contact'" @close="close">
    <form class="contact-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="role" label="Role" :options="roleOptions" />
      <BaseInput v-model="email" label="Email" />
      <BaseInput v-model="phone" label="Phone" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="contact-form__error" role="alert">{{ error }}</p>

      <div class="contact-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : contact ? 'Save Changes' : 'Add Contact' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.contact-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.contact-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.contact-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
