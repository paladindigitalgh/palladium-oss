<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { listDevices } from '@/services/devices/deviceRepository'
import { listCustomerDevices, attachCustomerDevice } from '@/services/customerDevices/customerDeviceRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { CustomerDevice } from '@/types/customerDevice'
import type { Location } from '@/types/location'

/**
 * Attaches an existing Device to a Customer, creating a CustomerDevice
 * record (internal/customerdevice) -- placing it at that Customer's
 * premises independent of any Service, mirroring
 * AssignServiceEquipmentDialog.vue one domain up. Create-only, the same
 * scope that dialog documents for its own equipment assignment: there is
 * no "reassign" flow here, only "attach this Device to this Customer
 * starting now."
 *
 * The Device picker is narrower than AssignServiceEquipmentDialog's:
 * Unused status alone is not enough here, since Status only tracks
 * Service attachment (see types/device.ts), not Customer attachment -- a
 * Device already attached to a different Customer can still be Unused.
 * So this fetches every CustomerDevice record too and excludes any
 * Device with an active one, regardless of status, on top of the
 * existing Unused filter. The backend enforces the same rule server-side
 * (internal/customerdevice/service.CustomerDeviceService.Create) as the
 * final authority; this is the cheap client-side narrowing that keeps
 * the picker from offering an option that would just be rejected.
 *
 * Location is optional tracking only (internal/customerdevice's own doc
 * comment on CustomerDevice.LocationID): which of the Customer's own
 * Locations this Device physically sits at, never read by any
 * provisioning or billing logic. `locations` is the parent's already-
 * loaded list for this Customer (CustomerDetailView.vue's own `locations`
 * ref) -- fetched once up there rather than duplicated here, the same
 * reasoning DeviceFormDialog.vue's Rack picker gives for reusing an
 * already-loaded list instead of fetching its own copy when one is cheaply
 * at hand.
 */
const props = defineProps<{ open: boolean; customerId: string; locations: Location[] }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'attached', record: CustomerDevice): void
}>()

const deviceId = ref('')
const locationId = ref('')
const availableDevices = ref<Device[]>([])
const submitting = ref(false)
const error = ref<string | null>(null)

const deviceOptions = computed(() =>
  availableDevices.value.map((device) => ({
    value: device.id,
    label: `${device.name} — ${device.manufacturer} ${device.model} (${device.serialNumber})`,
  })),
)

const locationOptions = computed(() => [
  { value: '', label: 'Not set' },
  ...props.locations.map((location) => ({ value: location.id, label: location.name })),
])

function reset() {
  deviceId.value = ''
  locationId.value = ''
  error.value = null
}

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    reset()
    const [unusedDevices, customerDevices] = await Promise.all([
      listDevices({ status: 'Unused', pageSize: 200 }),
      listCustomerDevices(),
    ])
    const attachedElsewhere = new Set(customerDevices.filter((r) => r.detachedAt === null).map((r) => r.deviceId))
    availableDevices.value = unusedDevices.items.filter((d) => !attachedElsewhere.has(d.id))
    deviceId.value = availableDevices.value[0]?.id ?? ''
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
    const record = await attachCustomerDevice({
      customerId: props.customerId,
      deviceId: deviceId.value,
      locationId: locationId.value === '' ? null : locationId.value,
    })
    reset()
    emit('attached', record)
  } catch (err) {
    error.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'That device is already attached to another customer — detach it there first.'
        : 'The device could not be attached.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Attach Device" @close="close">
    <form class="attach-form" @submit.prevent="handleSubmit">
      <p v-if="availableDevices.length === 0" class="attach-form__error" role="alert">
        No available devices — every Unused device is already attached to a customer. Add a device in Devices first.
      </p>
      <template v-else>
        <BaseSelect v-model="deviceId" label="Device" :options="deviceOptions" />
        <BaseSelect v-model="locationId" label="Location" :options="locationOptions" />
      </template>

      <p v-if="error" class="attach-form__error" role="alert">{{ error }}</p>

      <div class="attach-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton
          v-if="availableDevices.length > 0"
          type="submit"
          variant="primary"
          :disabled="submitting || !deviceId"
        >
          {{ submitting ? 'Attaching…' : 'Attach Device' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.attach-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.attach-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.attach-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
