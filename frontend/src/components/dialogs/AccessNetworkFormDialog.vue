<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createAccessNetwork, updateAccessNetwork } from '@/services/accessNetworks/accessNetworkRepository'
import { ApiError } from '@/services/api/httpClient'
import type { AccessNetwork } from '@/types/accessNetwork'

/**
 * Dual-mode: create when `accessNetwork` is absent, edit when present --
 * mirrors DeviceFormDialog.vue/CustomerFormDialog.vue. No parent id or
 * hidden-field passthrough concerns -- AccessNetwork is the root of its
 * hierarchy.
 */
const props = defineProps<{ open: boolean; accessNetwork?: AccessNetwork | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', accessNetwork: AccessNetwork): void
  (event: 'updated', accessNetwork: AccessNetwork): void
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

function populateFrom(accessNetwork: AccessNetwork) {
  name.value = accessNetwork.name
  status.value = accessNetwork.status
  description.value = accessNetwork.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from
// `accessNetwork` when editing or blank when creating -- not just once
// on mount, since the same mounted dialog instance is reused across
// opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.accessNetwork) populateFrom(props.accessNetwork)
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
    if (props.accessNetwork) {
      const updated = await updateAccessNetwork(props.accessNetwork.id, {
        name: name.value,
        status: status.value,
        description: description.value,
      })
      emit('updated', updated)
    } else {
      const accessNetwork = await createAccessNetwork({
        name: name.value,
        status: status.value,
        description: description.value,
      })
      reset()
      emit('created', accessNetwork)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The access network could not be ${props.accessNetwork ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="accessNetwork ? 'Edit Access Network' : 'New Access Network'" @close="close">
    <form class="access-network-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="access-network-form__error" role="alert">{{ error }}</p>

      <div class="access-network-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : accessNetwork ? 'Save Changes' : 'Create Access Network' }}
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
