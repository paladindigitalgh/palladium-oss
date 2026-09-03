<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createDevice, updateDevice } from '@/services/devices/deviceRepository'
import { listRacks } from '@/services/racks/rackRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { Rack } from '@/types/rack'

/**
 * Dual-mode: create when `device` is absent, edit when present -- one
 * dialog rather than a near-duplicate EditDeviceDialog, since every field
 * below is shared between the two (CLAUDE.md, "avoid unnecessary
 * abstractions" cuts the other way here: two components would only
 * duplicate this form). Editable fields are everything an operator might
 * reasonably need to correct after the fact -- name, manufacturer, model,
 * serial number, asset tag, status, description, and (see below) rack.
 * Identity (id, createdAt/updatedAt) never was.
 */
const props = defineProps<{ open: boolean; device?: Device | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', device: Device): void
  (event: 'updated', device: Device): void
}>()

const name = ref('')
const manufacturer = ref('')
const model = ref('')
const serialNumber = ref('')
const assetTag = ref('')
const status = ref<Device['status']>('InStock')
const description = ref('')
const rackId = ref('')
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

const racks = ref<Rack[]>([])
const rackOptions = computed(() => [{ value: '', label: 'None' }, ...racks.value.map((rack) => ({ value: rack.id, label: rack.name }))])

// Racks are fetched fresh each time the dialog opens, the same reasoning
// AttachAccessAttachmentDialog.vue documents for ServiceEquipment -- no
// cache to keep fresh, and this dataset is small.
watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return
    racks.value = await listRacks()
  },
)

function reset() {
  name.value = ''
  manufacturer.value = ''
  model.value = ''
  serialNumber.value = ''
  assetTag.value = ''
  status.value = 'InStock'
  description.value = ''
  rackId.value = ''
  error.value = null
}

function populateFrom(device: Device) {
  name.value = device.name
  manufacturer.value = device.manufacturer
  model.value = device.model
  serialNumber.value = device.serialNumber
  assetTag.value = device.assetTag
  status.value = device.status
  description.value = device.description
  rackId.value = device.rackId ?? ''
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `device`
// when editing or blank when creating -- not just once on mount, since
// the same mounted dialog instance is reused across opens (e.g. editing
// two different devices in the same session without navigating away).
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.device) populateFrom(props.device)
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
    const selectedRackId = rackId.value === '' ? null : rackId.value
    if (props.device) {
      const updated = await updateDevice(props.device.id, {
        name: name.value,
        manufacturer: manufacturer.value,
        model: model.value,
        serialNumber: serialNumber.value,
        assetTag: assetTag.value,
        status: status.value,
        description: description.value,
        rackId: selectedRackId,
      })
      emit('updated', updated)
    } else {
      const device = await createDevice({
        name: name.value,
        manufacturer: manufacturer.value,
        model: model.value,
        serialNumber: serialNumber.value,
        assetTag: assetTag.value,
        status: status.value,
        description: description.value,
        rackId: selectedRackId,
      })
      reset()
      emit('created', device)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The device could not be ${props.device ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="device ? 'Edit Device' : 'New Device'" @close="close">
    <form class="device-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="manufacturer" label="Manufacturer" required />
      <BaseInput v-model="model" label="Model" required />
      <BaseInput v-model="serialNumber" label="Serial Number" required />
      <BaseInput v-model="assetTag" label="Asset Tag" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />
      <BaseSelect v-model="rackId" label="Rack" :options="rackOptions" />

      <p v-if="error" class="device-form__error" role="alert">{{ error }}</p>

      <div class="device-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : device ? 'Save Changes' : 'Create Device' }}
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
