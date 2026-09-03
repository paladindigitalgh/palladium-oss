<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { updateAccessAttachment } from '@/services/accessAttachments/accessAttachmentRepository'
import { ApiError } from '@/services/api/httpClient'
import type { AccessAttachment } from '@/types/accessAttachment'

/**
 * Detaching is a PUT setting removedAt/removalReason, not a DELETE (see
 * accessAttachmentRepository.ts's own doc comment -- the row stays as
 * history). Single-purpose, populate-on-open shape like
 * DeviceFormDialog.vue's edit mode, but with no dual mode of its own:
 * this dialog only ever detaches the `attachment` it is given.
 */
const props = defineProps<{ open: boolean; attachment: AccessAttachment }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'detached', accessAttachment: AccessAttachment): void
}>()

const removalReason = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

watch(
  () => props.open,
  (isOpen) => {
    if (!isOpen) return
    removalReason.value = ''
    error.value = null
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
    const updated = await updateAccessAttachment(props.attachment.id, {
      accessInterfaceId: props.attachment.accessInterfaceId,
      serviceEquipmentId: props.attachment.serviceEquipmentId,
      installedAt: props.attachment.installedAt,
      removedAt: new Date().toISOString(),
      removalReason: removalReason.value,
    })
    emit('detached', updated)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The equipment could not be detached.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="Detach Equipment" @close="close">
    <form class="detach-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="removalReason" label="Reason" />

      <p v-if="error" class="detach-form__error" role="alert">{{ error }}</p>

      <div class="detach-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="destructive" :disabled="submitting">
          {{ submitting ? 'Detaching…' : 'Detach Equipment' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.detach-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.detach-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.detach-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
