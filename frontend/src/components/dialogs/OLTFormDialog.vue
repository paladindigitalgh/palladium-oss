<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createOLT, updateOLT } from '@/services/olts/oltRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLT } from '@/types/olt'

/**
 * Dual-mode: create when `olt` is absent, edit when present -- mirrors
 * DeviceFormDialog.vue. `accessNetworkId` (the parent prop, needed for
 * create) is ignored in edit mode -- the OLT being edited already has
 * one, and moving an OLT to a different Access Network is a bigger
 * operation than this dialog does. connectionProfileId is never a form
 * field (no picker exists) and is passed through unchanged on update,
 * the same reasoning as DeviceFormDialog.vue's rackId passthrough.
 */
const props = defineProps<{ open: boolean; accessNetworkId: string; olt?: OLT | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', olt: OLT): void
  (event: 'updated', olt: OLT): void
}>()

const name = ref('')
const vendor = ref<OLT['vendor']>('Nokia')
const model = ref('')
const managementIpAddress = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const vendorOptions = [
  { value: 'Kontron', label: 'Kontron' },
  { value: 'Nokia', label: 'Nokia' },
  { value: 'Calix', label: 'Calix' },
  { value: 'Adtran', label: 'Adtran' },
  { value: 'Other', label: 'Other' },
]

function reset() {
  name.value = ''
  vendor.value = 'Nokia'
  model.value = ''
  managementIpAddress.value = ''
  description.value = ''
  error.value = null
}

function populateFrom(olt: OLT) {
  name.value = olt.name
  vendor.value = olt.vendor
  model.value = olt.model
  managementIpAddress.value = olt.managementIpAddress
  description.value = olt.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `olt` when
// editing or blank when creating -- not just once on mount, since the
// same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.olt) populateFrom(props.olt)
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
    if (props.olt) {
      const updated = await updateOLT(props.olt.id, {
        name: name.value,
        vendor: vendor.value,
        model: model.value,
        managementIpAddress: managementIpAddress.value,
        description: description.value,
        accessNetworkId: props.olt.accessNetworkId,
        connectionProfileId: props.olt.connectionProfileId,
      })
      emit('updated', updated)
    } else {
      const olt = await createOLT({
        accessNetworkId: props.accessNetworkId,
        name: name.value,
        vendor: vendor.value,
        model: model.value,
        managementIpAddress: managementIpAddress.value,
        description: description.value,
      })
      reset()
      emit('created', olt)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The OLT could not be ${props.olt ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="olt ? 'Edit OLT' : 'Add OLT'" @close="close">
    <form class="olt-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="vendor" label="Vendor" :options="vendorOptions" />
      <BaseInput v-model="model" label="Model" />
      <BaseInput v-model="managementIpAddress" label="Management IP Address" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="olt-form__error" role="alert">{{ error }}</p>

      <div class="olt-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : olt ? 'Save Changes' : 'Add OLT' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.olt-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.olt-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.olt-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
