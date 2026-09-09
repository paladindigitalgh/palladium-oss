<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createDeviceManufacturer } from '@/services/deviceManufacturers/deviceManufacturerRepository'
import { ApiError } from '@/services/api/httpClient'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'

/**
 * Creates a DeviceManufacturer -- an Administration-managed catalog
 * entry naming a manufacturer of Device hardware (see
 * internal/devicemanufacturer's own doc comment). Create-only, mirroring
 * OLTModelFormDialog.vue: there is no "edit a device manufacturer" flow
 * yet, only "add a new one" -- correcting a mistake means deleting and
 * recreating the entry, which the RESTRICT foreign key on
 * device_models.manufacturer_id already blocks once a real DeviceModel
 * references it.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', manufacturer: DeviceManufacturer): void
}>()

const name = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

function reset() {
  name.value = ''
  description.value = ''
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
    const manufacturer = await createDeviceManufacturer({ name: name.value, description: description.value })
    reset()
    emit('created', manufacturer)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The device manufacturer could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Device Manufacturer" @close="close">
    <form class="device-manufacturer-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" placeholder="Nokia" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="device-manufacturer-form__error" role="alert">{{ error }}</p>

      <div class="device-manufacturer-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name">
          {{ submitting ? 'Creating…' : 'Create Device Manufacturer' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.device-manufacturer-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.device-manufacturer-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.device-manufacturer-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
