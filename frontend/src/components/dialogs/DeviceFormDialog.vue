<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createDevice, updateDevice } from '@/services/devices/deviceRepository'
import { listRacks } from '@/services/racks/rackRepository'
import { listDeviceManufacturers } from '@/services/deviceManufacturers/deviceManufacturerRepository'
import { listDeviceModels } from '@/services/deviceModels/deviceModelRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { Rack } from '@/types/rack'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'
import type { DeviceModel } from '@/types/deviceModel'

/**
 * Dual-mode: create when `device` is absent, edit when present -- one
 * dialog rather than a near-duplicate EditDeviceDialog, since every field
 * below is shared between the two (CLAUDE.md, "avoid unnecessary
 * abstractions" cuts the other way here: two components would only
 * duplicate this form). Editable fields are everything an operator might
 * reasonably need to correct after the fact -- name, manufacturer, model,
 * serial number, asset tag, status, description, and (see below) rack.
 * Identity (id, createdAt/updatedAt) never was.
 *
 * Manufacturer and Model are cascading pickers, not free text: an
 * operator selects a Device Manufacturer (internal/devicemanufacturer),
 * which narrows the Model picker to that manufacturer's Device Models
 * (internal/devicemodel) -- see those catalogs' own doc comments for why
 * Device stopped carrying these as free-text strings directly. The
 * Manufacturer picker itself is a UI-only concept for narrowing the
 * Model list; only manufacturerId is used to filter modelOptions, and
 * only modelId (as deviceModelId) is ever sent to the backend.
 */
const props = defineProps<{ open: boolean; device?: Device | null; initialSerialNumber?: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', device: Device): void
  (event: 'updated', device: Device): void
}>()

const name = ref('')
const manufacturerId = ref('')
const modelId = ref('')
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

const manufacturers = ref<DeviceManufacturer[]>([])
const manufacturerOptions = computed(() => manufacturers.value.map((m) => ({ value: m.id, label: m.name })))

const models = ref<DeviceModel[]>([])
const modelOptions = computed(() =>
  models.value.filter((m) => m.manufacturerId === manufacturerId.value).map((m) => ({ value: m.id, label: m.name })),
)

// Switching Manufacturer clears Model whenever it no longer belongs to
// the newly-selected Manufacturer. This only guards the user actively
// changing the Manufacturer picker: the fetch-then-populate watcher
// below sets manufacturerId and modelId together correctly on every
// open, and by the time this watcher's callback runs (deferred to the
// next microtask flush, same as every Vue watcher), both assignments
// have already happened -- so it sees a modelId that already belongs to
// the just-set manufacturerId and leaves it alone.
watch(manufacturerId, () => {
  if (!models.value.some((m) => m.id === modelId.value && m.manufacturerId === manufacturerId.value)) {
    modelId.value = ''
  }
})

function reset() {
  name.value = ''
  manufacturerId.value = ''
  modelId.value = ''
  serialNumber.value = props.initialSerialNumber ?? ''
  assetTag.value = ''
  status.value = 'InStock'
  description.value = ''
  rackId.value = ''
  error.value = null
}

function populateFrom(device: Device) {
  name.value = device.name
  modelId.value = device.deviceModelId
  manufacturerId.value = models.value.find((m) => m.id === device.deviceModelId)?.manufacturerId ?? ''
  serialNumber.value = device.serialNumber
  assetTag.value = device.assetTag
  status.value = device.status
  description.value = device.description
  rackId.value = device.rackId ?? ''
  error.value = null
}

// Racks, Device Manufacturers, and Device Models are fetched fresh each
// time the dialog opens, the same reasoning AttachAccessAttachmentDialog.vue
// documents for ServiceEquipment -- no cache to keep fresh, and these
// datasets are small. Fields are (re)populated in this same watcher,
// after the fetch resolves -- not just once on mount, since the same
// mounted dialog instance is reused across opens (e.g. editing two
// different devices in the same session without navigating away) --
// the same one-watcher-not-two ordering OLTFormDialog.vue's own
// OLTModel/ConnectionProfile fetch documents, so populateFrom's
// models.value lookup is guaranteed to see the freshly-fetched list,
// never a stale one from a race between two independent watchers.
watch(
  () => props.open,
  async (open) => {
    if (!open) return
    ;[racks.value, manufacturers.value, models.value] = await Promise.all([listRacks(), listDeviceManufacturers(), listDeviceModels()])
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
        deviceModelId: modelId.value,
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
        deviceModelId: modelId.value,
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
      <BaseSelect v-model="manufacturerId" label="Manufacturer" :options="manufacturerOptions" />
      <BaseSelect v-model="modelId" label="Model" :options="modelOptions" />
      <BaseInput v-model="serialNumber" label="Serial Number" required />
      <BaseInput v-model="assetTag" label="Asset Tag" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />
      <BaseSelect v-model="rackId" label="Rack" :options="rackOptions" />

      <p v-if="error" class="device-form__error" role="alert">{{ error }}</p>

      <div class="device-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !modelId">
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
