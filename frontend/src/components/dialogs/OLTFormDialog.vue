<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createOLT, updateOLT } from '@/services/olts/oltRepository'
import { listOLTModels } from '@/services/oltModels/oltModelRepository'
import { listConnectionProfiles } from '@/services/connectionProfiles/connectionProfileRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLT } from '@/types/olt'
import type { OLTModel } from '@/types/oltModel'
import type { ConnectionProfile } from '@/types/connectionProfile'

/**
 * Dual-mode: create when `olt` is absent, edit when present -- mirrors
 * DeviceFormDialog.vue. `accessNetworkId` (the parent prop, needed for
 * create) is ignored in edit mode -- the OLT being edited already has
 * one, and moving an OLT to a different Access Network is a bigger
 * operation than this dialog does.
 *
 * The OLT Model picker replaces this form's former free-text Vendor and
 * Model fields (see internal/olt/model.go's package doc comment on why
 * both moved to the OLTModel catalog). The Connection Profile picker is
 * nullable and defaults to "None" (empty-string sentinel, converted to
 * null on submit), the same convention DeviceFormDialog.vue's Rack
 * picker uses -- an OLT can exist with no Connection Profile, e.g.
 * before one has been created. OLTModels and ConnectionProfiles are
 * both fetched fresh each time the dialog opens, the same reasoning
 * DeviceFormDialog.vue's own Rack picker documents -- no cache to keep
 * fresh, and both datasets are small.
 */
const props = defineProps<{ open: boolean; accessNetworkId: string; olt?: OLT | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', olt: OLT): void
  (event: 'updated', olt: OLT): void
}>()

const name = ref('')
const oltModelId = ref('')
const managementIpAddress = ref('')
const connectionProfileId = ref('')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const oltModels = ref<OLTModel[]>([])
const oltModelOptions = computed(() =>
  oltModels.value.map((model) => ({ value: model.id, label: `${model.vendor} ${model.name}` })),
)

const connectionProfiles = ref<ConnectionProfile[]>([])
const connectionProfileOptions = computed(() => [
  { value: '', label: 'None' },
  ...connectionProfiles.value.map((profile) => ({ value: profile.id, label: profile.name })),
])

function reset() {
  name.value = ''
  oltModelId.value = ''
  managementIpAddress.value = ''
  connectionProfileId.value = ''
  description.value = ''
  error.value = null
}

function populateFrom(olt: OLT) {
  name.value = olt.name
  oltModelId.value = olt.oltModelId
  managementIpAddress.value = olt.managementIpAddress
  connectionProfileId.value = olt.connectionProfileId ?? ''
  description.value = olt.description
  error.value = null
}

// Fields are (re)populated every time the dialog opens, from `olt` when
// editing or blank when creating -- not just once on mount, since the
// same mounted dialog instance is reused across opens. OLTModels and
// ConnectionProfiles are fetched in this same watcher, before
// populate/reset run, so create mode's default selection (the first
// fetched model) is never racing the fetch itself -- the same reasoning
// DeviceFormDialog.vue's own Rack picker documents for fetching fresh on
// every open, applied here with one watcher instead of two to keep that
// ordering guaranteed rather than incidental.
watch(
  () => props.open,
  async (open) => {
    if (!open) return
    ;[oltModels.value, connectionProfiles.value] = await Promise.all([listOLTModels(), listConnectionProfiles()])
    if (props.olt) {
      populateFrom(props.olt)
    } else {
      reset()
      oltModelId.value = oltModels.value[0]?.id ?? ''
    }
  },
  { immediate: true },
)

function close() {
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  const selectedConnectionProfileId = connectionProfileId.value === '' ? null : connectionProfileId.value
  try {
    if (props.olt) {
      const updated = await updateOLT(props.olt.id, {
        name: name.value,
        oltModelId: oltModelId.value,
        managementIpAddress: managementIpAddress.value,
        description: description.value,
        accessNetworkId: props.olt.accessNetworkId,
        connectionProfileId: selectedConnectionProfileId,
      })
      emit('updated', updated)
    } else {
      const olt = await createOLT({
        accessNetworkId: props.accessNetworkId,
        name: name.value,
        oltModelId: oltModelId.value,
        managementIpAddress: managementIpAddress.value,
        description: description.value,
        connectionProfileId: selectedConnectionProfileId,
      })
      reset()
      emit('created', olt)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The OLT could not be ${props.olt ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="olt ? 'Edit OLT' : 'Add OLT'" @close="close">
    <form class="olt-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" required />
      <BaseSelect v-model="oltModelId" label="OLT Model" :options="oltModelOptions" />
      <BaseInput v-model="managementIpAddress" label="Management IP Address" />
      <BaseSelect v-model="connectionProfileId" label="Connection Profile" :options="connectionProfileOptions" />
      <BaseInput v-model="description" label="Description" />

      <p v-if="error" class="olt-form__error" role="alert">{{ error }}</p>

      <div class="olt-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name || !oltModelId">
          {{ submitting ? 'Saving…' : olt ? 'Save Changes' : 'Add OLT' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.olt-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.olt-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.olt-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
