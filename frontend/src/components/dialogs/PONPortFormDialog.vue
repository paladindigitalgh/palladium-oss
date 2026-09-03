<script setup lang="ts">
import { ref } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createPONPort } from '@/services/ponPorts/ponPortRepository'
import { ApiError } from '@/services/api/httpClient'
import type { PONPort } from '@/types/ponPort'

const props = defineProps<{ open: boolean; oltId: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', ponPort: PONPort): void
}>()

const portNumber = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

function reset() {
  portNumber.value = ''
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
    const ponPort = await createPONPort({
      oltId: props.oltId,
      portNumber: Number(portNumber.value),
      description: description.value,
    })
    reset()
    emit('created', ponPort)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The PON port could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Add PON Port" @close="close">
    <form class="pon-port-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="portNumber" label="Port Number" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="pon-port-form__error" role="alert">{{ error }}</p>

      <div class="pon-port-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Adding…' : 'Add PON Port' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.pon-port-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.pon-port-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.pon-port-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
