<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createDeviceModel } from '@/services/deviceModels/deviceModelRepository'
import { listDeviceManufacturers } from '@/services/deviceManufacturers/deviceManufacturerRepository'
import { ApiError } from '@/services/api/httpClient'
import type { DeviceModel } from '@/types/deviceModel'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'

/**
 * Creates a DeviceModel -- an Administration-managed catalog entry
 * naming one hardware model made by a DeviceManufacturer (see
 * internal/devicemodel's own doc comment). Create-only, mirroring
 * OLTModelFormDialog.vue and DeviceManufacturerFormDialog.vue: there is
 * no "edit a device model" flow yet, only "add a new one" -- correcting
 * a mistake means deleting and recreating the entry, which the RESTRICT
 * foreign key on devices.device_model_id already blocks once a real
 * Device references it.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', model: DeviceModel): void
}>()

const manufacturerId = ref('')
const name = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const manufacturers = ref<DeviceManufacturer[]>([])
const manufacturerOptions = computed(() => manufacturers.value.map((m) => ({ value: m.id, label: m.name })))

function reset() {
  manufacturerId.value = manufacturers.value[0]?.id ?? ''
  name.value = ''
  description.value = ''
  error.value = null
}

// Manufacturers are fetched fresh each time the dialog opens, the same
// reasoning OLTFormDialog.vue's own Connection Profile picker documents
// -- no cache to keep fresh, and this dataset is small. Fetched before
// reset() runs so its default selection (the first fetched manufacturer)
// is never racing the fetch itself.
watch(
  () => props.open,
  async (open) => {
    if (!open) return
    manufacturers.value = await listDeviceManufacturers()
    reset()
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
    const model = await createDeviceModel({
      manufacturerId: manufacturerId.value,
      name: name.value,
      description: description.value,
    })
    reset()
    emit('created', model)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The device model could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Device Model" @close="close">
    <form class="device-model-form" @submit.prevent="handleSubmit">
      <BaseSelect v-model="manufacturerId" label="Manufacturer" :options="manufacturerOptions" />
      <BaseInput v-model="name" label="Name" placeholder="G-140W-CT" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="device-model-form__error" role="alert">{{ error }}</p>

      <div class="device-model-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name || !manufacturerId">
          {{ submitting ? 'Creating…' : 'Create Device Model' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.device-model-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.device-model-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.device-model-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
