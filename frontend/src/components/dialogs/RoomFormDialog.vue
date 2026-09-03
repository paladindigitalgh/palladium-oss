<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createRoom, updateRoom } from '@/services/rooms/roomRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Room } from '@/types/room'

/**
 * Dual-mode: create when `room` is absent, edit when present -- mirrors
 * BuildingFormDialog.vue. `buildingId` (the parent prop, needed for
 * create) is ignored in edit mode -- the Room being edited already has
 * one, and moving a Room to a different Building is a bigger operation
 * than this dialog does.
 */
const props = defineProps<{ open: boolean; buildingId: string; room?: Room | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', room: Room): void
  (event: 'updated', room: Room): void
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

function populateFrom(room: Room) {
  name.value = room.name
  description.value = room.description
  error.value = null
}

watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.room) populateFrom(props.room)
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
    if (props.room) {
      const updated = await updateRoom(props.room.id, {
        name: name.value,
        description: description.value,
        buildingId: props.room.buildingId,
      })
      emit('updated', updated)
    } else {
      const room = await createRoom({ buildingId: props.buildingId, name: name.value, description: description.value })
      reset()
      emit('created', room)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The room could not be ${props.room ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="room ? 'Edit Room' : 'Add Room'" @close="close">
    <form class="room-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="room-form__error" role="alert">{{ error }}</p>

      <div class="room-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : room ? 'Save Changes' : 'Add Room' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.room-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.room-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.room-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
