<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createCustomer, updateCustomer } from '@/services/customers/customerRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Customer } from '@/types/customer'

/**
 * Dual-mode: create when `customer` is absent, edit when present --
 * mirrors DeviceFormDialog.vue exactly, the established pattern for
 * every *FormDialog that needs both. Every field is editable; only
 * identity (id, createdAt/updatedAt) never was.
 */
const props = defineProps<{ open: boolean; customer?: Customer | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', customer: Customer): void
  (event: 'updated', customer: Customer): void
}>()

const name = ref('')
const customerType = ref<Customer['customerType']>('Residential')
const status = ref<Customer['status']>('Active')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const typeOptions = [
  { value: 'Residential', label: 'Residential' },
  { value: 'Business', label: 'Business' },
  { value: 'Government', label: 'Government' },
  { value: 'Internal', label: 'Internal' },
]

const statusOptions = [
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
  { value: 'Archived', label: 'Archived' },
]

function reset() {
  name.value = ''
  customerType.value = 'Residential'
  status.value = 'Active'
  description.value = ''
  error.value = null
}

function populateFrom(customer: Customer) {
  name.value = customer.name
  customerType.value = customer.customerType
  status.value = customer.status
  description.value = customer.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `customer`
// when editing or blank when creating -- not just once on mount, since
// the same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.customer) populateFrom(props.customer)
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
    if (props.customer) {
      const updated = await updateCustomer(props.customer.id, {
        name: name.value,
        customerType: customerType.value,
        status: status.value,
        description: description.value,
      })
      emit('updated', updated)
    } else {
      const customer = await createCustomer({
        name: name.value,
        customerType: customerType.value,
        status: status.value,
        description: description.value,
      })
      reset()
      emit('created', customer)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The customer could not be ${props.customer ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="customer ? 'Edit Customer' : 'New Customer'" @close="close">
    <form class="customer-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="customerType" label="Customer Type" :options="typeOptions" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="customer-form__error" role="alert">{{ error }}</p>

      <div class="customer-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : customer ? 'Save Changes' : 'Create Customer' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.customer-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.customer-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.customer-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
