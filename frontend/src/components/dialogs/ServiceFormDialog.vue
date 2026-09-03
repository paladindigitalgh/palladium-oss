<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createService, updateService } from '@/services/services/serviceRepository'
import { listProducts } from '@/services/products/productRepository'
import { listServiceProfiles } from '@/services/serviceProfiles/serviceProfileRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Service } from '@/types/service'
import type { Product } from '@/types/product'
import type { ServiceProfile } from '@/types/serviceProfile'

/**
 * Dual-mode: create when `service` is absent, edit when present --
 * mirrors DeviceFormDialog.vue/CustomerFormDialog.vue. `locationId` (the
 * parent prop, needed for create since a new Service has no location of
 * its own yet) is ignored in edit mode -- the service being edited
 * already has one, and moving a Service to a different Location is a
 * bigger operation than this dialog does. activatedAt/suspendedAt/
 * disconnectedAt are never form fields (see serviceRepository.ts's
 * UpdateServiceInput doc comment: those belong to the Workflow Engine).
 */
const props = defineProps<{ open: boolean; locationId: string; service?: Service | null }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', service: Service): void
  (event: 'updated', service: Service): void
}>()

const products = ref<Product[]>([])
const serviceProfiles = ref<ServiceProfile[]>([])
const loadingOptions = ref(false)

const productId = ref('')
const serviceProfileId = ref('')
const status = ref<Service['status']>('Pending')
const description = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const statusOptions = [
  { value: 'Pending', label: 'Pending' },
  { value: 'Active', label: 'Active' },
  { value: 'Suspended', label: 'Suspended' },
  { value: 'Disconnected', label: 'Disconnected' },
]

// Products/Service Profiles are fetched fresh each time the dialog opens
// rather than once at app startup -- there is no Product/Service Profile
// Workspace to keep a cached copy fresh against, and this dataset is
// small enough that refetching is simpler than inventing a cache to
// invalidate. Needed in both modes: create defaults to the first option,
// edit needs the options list to show the service's current selection.
watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return
    error.value = null
    loadingOptions.value = true
    const [productList, profileList] = await Promise.all([listProducts(), listServiceProfiles()])
    products.value = productList
    serviceProfiles.value = profileList

    if (props.service) {
      productId.value = props.service.productId
      serviceProfileId.value = props.service.serviceProfileId
      status.value = props.service.status
      description.value = props.service.description
    } else {
      productId.value = productList[0]?.id ?? ''
      serviceProfileId.value = profileList[0]?.id ?? ''
      status.value = 'Pending'
      description.value = ''
    }

    loadingOptions.value = false
  },
)

function close() {
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
    if (props.service) {
      const updated = await updateService(props.service.id, {
        locationId: props.service.locationId,
        productId: productId.value,
        serviceProfileId: serviceProfileId.value,
        status: status.value,
        description: description.value,
        activatedAt: props.service.activatedAt,
        suspendedAt: props.service.suspendedAt,
        disconnectedAt: props.service.disconnectedAt,
      })
      emit('updated', updated)
    } else {
      const service = await createService({
        locationId: props.locationId,
        productId: productId.value,
        serviceProfileId: serviceProfileId.value,
        status: status.value,
        description: description.value,
      })
      emit('created', service)
    }
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : `The service could not be ${props.service ? 'saved' : 'created'}.`
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" :title="service ? 'Edit Service' : 'Add Service'" @close="close">
    <p v-if="loadingOptions" class="service-form__loading">Loading products and service profiles…</p>

    <form v-else class="service-form" @submit.prevent="handleSubmit">
      <p v-if="products.length === 0" class="service-form__error" role="alert">
        No products exist yet — create one in the catalog before adding a service.
      </p>
      <p v-else-if="serviceProfiles.length === 0" class="service-form__error" role="alert">
        No service profiles exist yet — create one before adding a service.
      </p>
      <template v-else>
        <BaseSelect
          v-model="productId"
          label="Product"
          :options="products.map((p) => ({ value: p.id, label: p.name }))"
        />
        <BaseSelect
          v-model="serviceProfileId"
          label="Service Profile"
          :options="serviceProfiles.map((p) => ({ value: p.id, label: p.name }))"
        />
        <BaseSelect v-model="status" label="Status" :options="statusOptions" />
        <BaseInput v-model="description" label="Description" />
      </template>

      <p v-if="error" class="service-form__error" role="alert">{{ error }}</p>

      <div class="service-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton
          v-if="products.length > 0 && serviceProfiles.length > 0"
          type="submit"
          variant="primary"
          :disabled="submitting"
        >
          {{ submitting ? 'Saving…' : service ? 'Save Changes' : 'Add Service' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.service-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.service-form__loading {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.service-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.service-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
