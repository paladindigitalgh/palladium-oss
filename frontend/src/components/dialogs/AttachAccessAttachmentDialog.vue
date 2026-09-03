<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createAccessAttachment } from '@/services/accessAttachments/accessAttachmentRepository'
import { listServiceEquipment } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { ApiError } from '@/services/api/httpClient'
import type { AccessAttachment } from '@/types/accessAttachment'
import type { ServiceEquipment } from '@/types/serviceEquipment'

const props = defineProps<{ open: boolean; accessInterfaceId: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', accessAttachment: AccessAttachment): void
}>()

const equipment = ref<ServiceEquipment[]>([])
const loadingOptions = ref(false)

const serviceEquipmentId = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

// ServiceEquipment is fetched fresh each time the dialog opens, the same
// reasoning ServiceFormDialog.vue documents for Products/Service
// Profiles -- no cache to keep fresh, and this dataset is small.
watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return
    loadingOptions.value = true
    const equipmentList = await listServiceEquipment()
    equipment.value = equipmentList
    serviceEquipmentId.value = equipmentList[0]?.id ?? ''
    loadingOptions.value = false
  },
)

function reset() {
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
    const accessAttachment = await createAccessAttachment({
      accessInterfaceId: props.accessInterfaceId,
      serviceEquipmentId: serviceEquipmentId.value,
    })
    reset()
    emit('created', accessAttachment)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The equipment could not be attached.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Attach Equipment" @close="close">
    <p v-if="loadingOptions" class="attach-form__loading">Loading service equipment…</p>

    <form v-else class="attach-form" @submit.prevent="handleSubmit">
      <p v-if="equipment.length === 0" class="attach-form__error" role="alert">
        No service equipment exists yet — assign equipment to a service before attaching it here.
      </p>
      <BaseSelect
        v-else
        v-model="serviceEquipmentId"
        label="Equipment"
        :options="equipment.map((item) => ({ value: item.id, label: `${item.role} — ${item.deviceId}` }))"
      />

      <p v-if="error" class="attach-form__error" role="alert">{{ error }}</p>

      <div class="attach-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton v-if="equipment.length > 0" type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Attaching…' : 'Attach Equipment' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.attach-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.attach-form__loading {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.attach-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.attach-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
