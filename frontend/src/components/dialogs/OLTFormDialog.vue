<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createOLT } from '@/services/olts/oltRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLT } from '@/types/olt'

const props = defineProps<{ open: boolean; accessNetworkId: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', olt: OLT): void
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

function close() {
  reset()
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
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
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The OLT could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Add OLT" @close="close">
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
          {{ submitting ? 'Adding…' : 'Add OLT' }}
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
