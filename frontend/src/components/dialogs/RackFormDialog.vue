<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createRack, updateRack } from '@/services/racks/rackRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Rack } from '@/types/rack'

/**
 * Dual-mode: create when `rack` is absent, edit when present -- mirrors
 * RoomFormDialog.vue. `roomId` (the parent prop, needed for create) is
 * ignored in edit mode -- the Rack being edited already has one (or
 * none), and moving a Rack to a different Room is a bigger operation
 * than this dialog does.
 */
const props = defineProps<{ open: boolean; roomId: string; rack?: Rack | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', rack: Rack): void
  (event: 'updated', rack: Rack): void
}>()

const name = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

function reset() {
  name.value = ''
  description.value = ''
  error.value = null
}

function populateFrom(rack: Rack) {
  name.value = rack.name
  description.value = rack.description
  error.value = null
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.rack) populateFrom(props.rack)
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
    if (props.rack) {
      const updated = await updateRack(props.rack.id, {
        name: name.value,
        description: description.value,
        roomId: props.rack.roomId,
      })
      emit('updated', updated)
    } else {
      const rack = await createRack({ roomId: props.roomId, name: name.value, description: description.value })
      reset()
      emit('created', rack)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The rack could not be ${props.rack ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="rack ? 'Edit Rack' : 'Add Rack'" @close="close">
    <form class="rack-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="rack-form__error" role="alert">{{ error }}</p>

      <div class="rack-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : rack ? 'Save Changes' : 'Add Rack' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.rack-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.rack-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.rack-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
