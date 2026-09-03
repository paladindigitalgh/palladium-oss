<script setup lang="ts">
import BaseModal from '@/components/base/BaseModal.vue'
import BaseButton from '@/components/base/BaseButton.vue'

/**
 * Every delete in the app goes through this (docs/07-UI-ARCHITECTURE.md
 * section 14: "Dialogs are used for focused, interruptive tasks
 * requiring explicit confirmation" -- "Delete confirmation" is the named
 * example). `error`, when set, is shown inline rather than closing the
 * dialog -- the caller keeps it open on failure (e.g. a 409 from a
 * foreign key restriction) so the operator sees why and can cancel with
 * full context, rather than losing the dialog and hunting for a toast.
 */
withDefaults(
  defineProps<{
    open: boolean
    title: string
    description: string
    confirmLabel?: string
    destructive?: boolean
    pending?: boolean
    error?: string | null
  }>(),
  { confirmLabel: 'Confirm', destructive: false, pending: false, error: null },
)

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()
</script>

<template>
  <BaseModal :open="open" :title="title" @close="emit('cancel')">
    <p class="confirmation-dialog__description">{{ description }}</p>
    <p v-if="error" class="confirmation-dialog__error" role="alert">{{ error }}</p>
    <div class="confirmation-dialog__actions">
      <BaseButton variant="secondary" :disabled="pending" @click="emit('cancel')">Cancel</BaseButton>
      <BaseButton :variant="destructive ? 'destructive' : 'primary'" :disabled="pending" @click="emit('confirm')">
        {{ pending ? 'Working…' : confirmLabel }}
      </BaseButton>
    </div>
  </BaseModal>
</template>

<style scoped>
.confirmation-dialog__description {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.confirmation-dialog__error {
  margin-top: var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.confirmation-dialog__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-5);
}
</style>
