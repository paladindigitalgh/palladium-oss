<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createProvider } from '@/services/providers/providerRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Provider } from '@/types/provider'

/**
 * Creates a Provider -- the retail ISP identity a Plan belongs to (see
 * internal/provider's own doc comment). Create-only, like
 * AssignServiceEquipmentDialog.vue: there is no "edit a provider" flow
 * yet, only "add a new one." In a single-ISP deployment this dialog is
 * used exactly once, to name the one Provider everything else belongs
 * to; AdministrationView.vue stops showing Provider pickers/columns
 * anywhere once that one Provider exists, so nothing here forces an
 * operator who will never have a second Provider to keep thinking about
 * it.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', provider: Provider): void
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
    const provider = await createProvider({ name: name.value, description: description.value })
    reset()
    emit('created', provider)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The provider could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Provider" @close="close">
    <form class="provider-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" placeholder="Acme Fiber" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="provider-form__error" role="alert">{{ error }}</p>

      <div class="provider-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name">
          {{ submitting ? 'Creating…' : 'Create Provider' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.provider-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.provider-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.provider-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
