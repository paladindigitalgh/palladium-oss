<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createOLTModel } from '@/services/oltModels/oltModelRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLTModel } from '@/types/oltModel'

/**
 * Creates an OLTModel -- an Administration-managed catalog entry naming
 * a physical OLT chassis type and how many PON ports it has (see
 * internal/oltmodel's own doc comment). Create-only, like
 * ProviderFormDialog.vue: there is no "edit an olt model" flow yet, only
 * "add a new one" -- correcting a mistake means deleting and recreating
 * the entry, which the RESTRICT foreign key on olts.olt_model_id already
 * blocks once a real OLT references it.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', model: OLTModel): void
}>()

const vendor = ref<OLTModel['vendor']>('Kontron')
const name = ref('')
// A string, not a number: BaseInput's model is always a string (see its
// own doc comment) -- converted to a number only where PONPortCount is
// actually consumed, in isValidPONPortCount and handleSubmit below.
const ponPortCountText = ref('16')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

function isValidPONPortCount(): boolean {
  const n = Number(ponPortCountText.value)
  return Number.isInteger(n) && n > 0
}

const vendorOptions = [
  { value: 'Kontron', label: 'Kontron' },
  { value: 'Nokia', label: 'Nokia' },
  { value: 'Calix', label: 'Calix' },
  { value: 'Adtran', label: 'Adtran' },
  { value: 'Other', label: 'Other' },
]

function reset() {
  vendor.value = 'Kontron'
  name.value = ''
  ponPortCountText.value = '16'
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
    const model = await createOLTModel({
      vendor: vendor.value,
      name: name.value,
      ponPortCount: Number(ponPortCountText.value),
      description: description.value,
    })
    reset()
    emit('created', model)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The OLT model could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New OLT Model" @close="close">
    <form class="olt-model-form" @submit.prevent="handleSubmit">
      <BaseSelect v-model="vendor" label="Vendor" :options="vendorOptions" />
      <BaseInput v-model="name" label="Name" placeholder="C16" required />
      <BaseInput v-model="ponPortCountText" label="PON Port Count" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="olt-model-form__error" role="alert">{{ error }}</p>

      <div class="olt-model-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name || !isValidPONPortCount()">
          {{ submitting ? 'Creating…' : 'Create OLT Model' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.olt-model-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.olt-model-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.olt-model-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
