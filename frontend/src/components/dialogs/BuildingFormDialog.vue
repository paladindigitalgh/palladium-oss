<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createBuilding, updateBuilding } from '@/services/buildings/buildingRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Building } from '@/types/building'

/**
 * Dual-mode: create when `building` is absent, edit when present --
 * mirrors OLTFormDialog.vue, minus the vendor/model/management-IP
 * fields OLT has and Building doesn't. `siteId` (the parent prop,
 * needed for create) is ignored in edit mode -- the Building being
 * edited already has one, and moving a Building to a different Site is
 * a bigger operation than this dialog does.
 */
const props = defineProps<{ open: boolean; siteId: string; building?: Building | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', building: Building): void
  (event: 'updated', building: Building): void
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

function populateFrom(building: Building) {
  name.value = building.name
  description.value = building.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `building`
// when editing or blank when creating -- not just once on mount, since
// the same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.building) populateFrom(props.building)
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
    if (props.building) {
      const updated = await updateBuilding(props.building.id, {
        name: name.value,
        description: description.value,
        siteId: props.building.siteId,
      })
      emit('updated', updated)
    } else {
      const building = await createBuilding({ siteId: props.siteId, name: name.value, description: description.value })
      reset()
      emit('created', building)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The building could not be ${props.building ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="building ? 'Edit Building' : 'Add Building'" @close="close">
    <form class="building-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="building-form__error" role="alert">{{ error }}</p>

      <div class="building-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : building ? 'Save Changes' : 'Add Building' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.building-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.building-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.building-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
