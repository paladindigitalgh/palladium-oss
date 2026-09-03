<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createAccessInterface, updateAccessInterface } from '@/services/accessInterfaces/accessInterfaceRepository'
import { ApiError } from '@/services/api/httpClient'
import type { AccessInterface } from '@/types/accessInterface'

/**
 * Dual-mode: create when `accessInterface` is absent, edit when present
 * -- mirrors DeviceFormDialog.vue. `ponPortId` (the parent prop, needed
 * for create) is ignored in edit mode -- the Access Interface being
 * edited already has one, and moving it to a different PON Port is a
 * bigger operation than this dialog does.
 */
const props = defineProps<{ open: boolean; ponPortId: string; accessInterface?: AccessInterface | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', accessInterface: AccessInterface): void
  (event: 'updated', accessInterface: AccessInterface): void
}>()

const technology = ref<AccessInterface['technology']>('GPON')
const name = ref('')
const status = ref<AccessInterface['status']>('Active')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const technologyOptions = [
  { value: 'GPON', label: 'GPON' },
  { value: 'XGSPON', label: 'XGS-PON' },
  { value: 'ActiveEthernet', label: 'Active Ethernet' },
  { value: 'Other', label: 'Other' },
]

const statusOptions = [
  { value: 'Active', label: 'Active' },
  { value: 'Disabled', label: 'Disabled' },
]

function reset() {
  technology.value = 'GPON'
  name.value = ''
  status.value = 'Active'
  description.value = ''
  error.value = null
}

function populateFrom(accessInterface: AccessInterface) {
  technology.value = accessInterface.technology
  name.value = accessInterface.name
  status.value = accessInterface.status
  description.value = accessInterface.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from
// `accessInterface` when editing or blank when creating -- not just
// once on mount, since the same mounted dialog instance is reused
// across opens.
watch(
  () => props.open,
  (open) => {
    if (!open) return
    if (props.accessInterface) populateFrom(props.accessInterface)
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
    if (props.accessInterface) {
      const updated = await updateAccessInterface(props.accessInterface.id, {
        technology: technology.value,
        name: name.value,
        status: status.value,
        description: description.value,
        ponPortId: props.accessInterface.ponPortId,
      })
      emit('updated', updated)
    } else {
      const accessInterface = await createAccessInterface({
        ponPortId: props.ponPortId,
        technology: technology.value,
        name: name.value,
        status: status.value,
        description: description.value,
      })
      reset()
      emit('created', accessInterface)
    }
  } catch (err) {
    error.value =
      err instanceof ApiError ? err.message : `The access interface could not be ${props.accessInterface ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="accessInterface ? 'Edit Access Interface' : 'Add Access Interface'" @close="close">
    <form class="access-interface-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="technology" label="Technology" :options="technologyOptions" />
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="access-interface-form__error" role="alert">{{ error }}</p>

      <div class="access-interface-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting">
          {{ submitting ? 'Saving…' : accessInterface ? 'Save Changes' : 'Add Access Interface' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.access-interface-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.access-interface-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.access-interface-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
