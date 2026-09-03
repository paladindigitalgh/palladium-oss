<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createPONPort, updatePONPort } from '@/services/ponPorts/ponPortRepository'
import { ApiError } from '@/services/api/httpClient'
import type { PONPort } from '@/types/ponPort'

/**
 * Dual-mode: create when `ponPort` is absent, edit when present --
 * mirrors DeviceFormDialog.vue. `oltId` (the parent prop, needed for
 * create) is ignored in edit mode -- the PON Port being edited already
 * has one, and moving a PON Port to a different OLT is a bigger
 * operation than this dialog does.
 */
const props = defineProps<{ open: boolean; oltId: string; ponPort?: PONPort | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', ponPort: PONPort): void
  (event: 'updated', ponPort: PONPort): void
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

function populateFrom(ponPort: PONPort) {
  portNumber.value = String(ponPort.portNumber)
  description.value = ponPort.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `ponPort`
// when editing or blank when creating -- not just once on mount, since
// the same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.ponPort) populateFrom(props.ponPort)
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
    if (props.ponPort) {
      const updated = await updatePONPort(props.ponPort.id, {
        portNumber: Number(portNumber.value),
        description: description.value,
        oltId: props.ponPort.oltId,
      })
      emit('updated', updated)
    } else {
      const ponPort = await createPONPort({
        oltId: props.oltId,
        portNumber: Number(portNumber.value),
        description: description.value,
      })
      reset()
      emit('created', ponPort)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The PON port could not be ${props.ponPort ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="ponPort ? 'Edit PON Port' : 'Add PON Port'" @close="close">
    <form class="pon-port-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="portNumber" label="Port Number" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="pon-port-form__error" role="alert">{{ error }}</p>

      <div class="pon-port-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : ponPort ? 'Save Changes' : 'Add PON Port' }}
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
