<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { listDevices } from '@/services/devices/deviceRepository'
import { createServiceEquipment } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { ServiceEquipment } from '@/types/serviceEquipment'

/**
 * Assigns an existing Device to a Service, creating a ServiceEquipment
 * record (docs/03-DOMAIN-MODEL.md section 7). Create-only, mirroring
 * this milestone's scope -- there is no "reassign" or "edit role" flow
 * here, only "attach this Device to this Service starting now" (see
 * serviceEquipmentRepository.ts's createServiceEquipment doc comment on
 * why installedAt/removedAt are not caller-editable fields on this
 * form). The Device picker follows the exact fetch-on-open pattern
 * DeviceFormDialog.vue's own Rack picker already establishes.
 */
const props = defineProps<{ open: boolean; serviceId: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', equipment: ServiceEquipment): void
}>()

const deviceId = ref('')
const role = ref<ServiceEquipment['role']>('ONU')
const description = ref('')
const devices = ref<Device[]>([])
const submitting = ref(false)
const error = ref<string | null>(null)

const deviceOptions = computed(() =>
  devices.value.map((device) => ({
    value: device.id,
    label: `${device.name} — ${device.manufacturer} ${device.model} (${device.serialNumber})`,
  })),
)

const roleOptions = [
  { value: 'ONU', label: 'ONU' },
  { value: 'ONT', label: 'ONT' },
  { value: 'Gateway', label: 'Gateway' },
  { value: 'Router', label: 'Router' },
  { value: 'WiFiAccessPoint', label: 'WiFi Access Point' },
  { value: 'UPS', label: 'UPS' },
  { value: 'Other', label: 'Other' },
]

function reset() {
  deviceId.value = ''
  role.value = 'ONU'
  description.value = ''
  error.value = null
}

// Devices are fetched fresh each time the dialog opens, the same
// reasoning AttachAccessAttachmentDialog.vue's ServiceEquipment picker
// documents -- and, like that picker, the first result becomes the
// default selection so the underlying <select> and deviceId never
// disagree about what is currently chosen (a blank '' with no matching
// option would leave the native element showing something the ref does
// not know about).
watch(
  () => props.open,
  async (open) => {
    if (!open) return
    reset()
    devices.value = (await listDevices({ pageSize: 200 })).items
    deviceId.value = devices.value[0]?.id ?? ''
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
    const equipment = await createServiceEquipment({
      serviceId: props.serviceId,
      deviceId: deviceId.value,
      role: role.value,
      description: description.value,
    })
    reset()
    emit('created', equipment)
  } catch (err) {
    error.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'That device is already assigned to another service — remove it there first.'
        : 'The device could not be assigned.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Assign Equipment" @close="close">
    <form class="assign-form" @submit.prevent="handleSubmit">
      <BaseSelect v-model="deviceId" label="Device" :options="deviceOptions" />
      <BaseSelect v-model="role" label="Role" :options="roleOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="assign-form__error" role="alert">{{ error }}</p>

      <div class="assign-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !deviceId">
          {{ submitting ? 'Assigning…' : 'Assign Equipment' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.assign-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.assign-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.assign-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
