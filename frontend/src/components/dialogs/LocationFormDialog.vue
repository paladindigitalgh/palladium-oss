<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createLocation } from '@/services/locations/locationRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Location } from '@/types/location'

const props = defineProps<{ open: boolean; customerId: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', location: Location): void
}>()

const name = ref('')
const type = ref<Location['type']>('Service')
const status = ref<Location['status']>('Active')
const address1 = ref('')
const address2 = ref('')
const city = ref('')
const state = ref('')
const postalCode = ref('')
const country = ref('US')
const submitting = ref(false)
const error = ref<string | null>(null)

const typeOptions = [
  { value: 'Service', label: 'Service' },
  { value: 'Billing', label: 'Billing' },
  { value: 'Office', label: 'Office' },
  { value: 'Warehouse', label: 'Warehouse' },
  { value: 'POP', label: 'POP' },
  { value: 'DataCenter', label: 'Data Center' },
  { value: 'Other', label: 'Other' },
]

const statusOptions = [
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
]

function reset() {
  name.value = ''
  type.value = 'Service'
  status.value = 'Active'
  address1.value = ''
  address2.value = ''
  city.value = ''
  state.value = ''
  postalCode.value = ''
  country.value = 'US'
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
    const location = await createLocation({
      customerId: props.customerId,
      name: name.value,
      type: type.value,
      status: status.value,
      address1: address1.value,
      address2: address2.value,
      city: city.value,
      state: state.value,
      postalCode: postalCode.value,
      country: country.value,
    })
    reset()
    emit('created', location)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The location could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Add Location" @close="close">
    <form class="location-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="type" label="Type" :options="typeOptions" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="address1" label="Address Line 1" />
      <BaseInput v-model="address2" label="Address Line 2" />
      <div class="location-form__row">
        <BaseInput v-model="city" label="City" />
        <BaseInput v-model="state" label="State" />
      </div>
      <div class="location-form__row">
        <BaseInput v-model="postalCode" label="Postal Code" />
        <BaseInput v-model="country" label="Country" />
      </div>

      <p v-if="error" class="location-form__error" role="alert">{{ error }}</p>

      <div class="location-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Adding…' : 'Add Location' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.location-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.location-form__row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.location-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.location-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
