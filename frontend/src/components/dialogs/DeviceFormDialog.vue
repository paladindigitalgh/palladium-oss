<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createDevice } from '@/services/devices/deviceRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'

defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', device: Device): void
}>()

const name = ref('')
const manufacturer = ref('')
const model = ref('')
const serialNumber = ref('')
const assetTag = ref('')
const status = ref<Device['status']>('InStock')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const statusOptions = [
  { value: 'Ordered', label: 'Ordered' },
  { value: 'Received', label: 'Received' },
  { value: 'InStock', label: 'In Stock' },
  { value: 'Installed', label: 'Installed' },
  { value: 'Maintenance', label: 'Maintenance' },
  { value: 'Retired', label: 'Retired' },
  { value: 'Disposed', label: 'Disposed' },
]

function reset() {
  name.value = ''
  manufacturer.value = ''
  model.value = ''
  serialNumber.value = ''
  assetTag.value = ''
  status.value = 'InStock'
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
    const device = await createDevice({
      name: name.value,
      manufacturer: manufacturer.value,
      model: model.value,
      serialNumber: serialNumber.value,
      assetTag: assetTag.value,
      status: status.value,
      description: description.value,
    })
    reset()
    emit('created', device)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The device could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Device" @close="close">
    <form class="device-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="manufacturer" label="Manufacturer" required />
      <BaseInput v-model="model" label="Model" required />
      <BaseInput v-model="serialNumber" label="Serial Number" required />
      <BaseInput v-model="assetTag" label="Asset Tag" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="device-form__error" role="alert">{{ error }}</p>

      <div class="device-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Creating…' : 'Create Device' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.device-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.device-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.device-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
