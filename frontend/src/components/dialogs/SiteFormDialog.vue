<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createSite, updateSite } from '@/services/sites/siteRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Site } from '@/types/site'

/**
 * Dual-mode: create when `site` is absent, edit when present -- mirrors
 * AccessNetworkFormDialog.vue, minus the status field (Site has none).
 * No parent id or hidden-field passthrough concerns -- Site is the root
 * of the Inventory hierarchy.
 */
const props = defineProps<{ open: boolean; site?: Site | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', site: Site): void
  (event: 'updated', site: Site): void
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

function populateFrom(site: Site) {
  name.value = site.name
  description.value = site.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `site` when
// editing or blank when creating -- not just once on mount, since the
// same mounted dialog instance is reused across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.site) populateFrom(props.site)
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
    if (props.site) {
      const updated = await updateSite(props.site.id, { name: name.value, description: description.value })
      emit('updated', updated)
    } else {
      const site = await createSite({ name: name.value, description: description.value })
      reset()
      emit('created', site)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The site could not be ${props.site ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="site ? 'Edit Site' : 'New Site'" @close="close">
    <form class="site-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="site-form__error" role="alert">{{ error }}</p>

      <div class="site-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : site ? 'Save Changes' : 'Create Site' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.site-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.site-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.site-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
