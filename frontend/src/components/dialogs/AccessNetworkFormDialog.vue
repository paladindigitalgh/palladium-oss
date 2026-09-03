<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createAccessNetwork } from '@/services/accessNetworks/accessNetworkRepository'
import { ApiError } from '@/services/api/httpClient'
import type { AccessNetwork } from '@/types/accessNetwork'

defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', accessNetwork: AccessNetwork): void
}>()

const name = ref('')
const status = ref<AccessNetwork['status']>('Active')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const statusOptions = [
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
]

function reset() {
  name.value = ''
  status.value = 'Active'
  description.value = ''
  error.value = null
}

function close() {
  reset()
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
    const accessNetwork = await createAccessNetwork({
      name: name.value,
      status: status.value,
      description: description.value,
    })
    reset()
    emit('created', accessNetwork)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The access network could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Access Network" @close="close">
    <form class="access-network-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="access-network-form__error" role="alert">{{ error }}</p>

      <div class="access-network-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Creating…' : 'Create Access Network' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.access-network-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.access-network-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.access-network-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
