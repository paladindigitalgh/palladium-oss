<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createService, updateService } from '@/services/services/serviceRepository'
import { createServiceEquipment } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { listProducts } from '@/services/products/productRepository'
import { listServiceProfiles } from '@/services/serviceProfiles/serviceProfileRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Service } from '@/types/service'
import type { Product } from '@/types/product'
import type { ServiceProfile } from '@/types/serviceProfile'
import type { Device } from '@/types/device'

/**
 * Dual-mode: create when `service` is absent, edit when present --
 * mirrors DeviceFormDialog.vue/CustomerFormDialog.vue. `locationId` (the
 * parent prop, needed for create since a new Service has no location of
 * its own yet) is ignored in edit mode -- the service being edited
 * already has one, and moving a Service to a different Location is a
 * bigger operation than this dialog does. activatedAt/suspendedAt/
 * disconnectedAt are never form fields (see serviceRepository.ts's
 * UpdateServiceInput doc comment: those belong to the Workflow Engine).
 *
 * `devices` is create-mode only, and this dialog's only caller in create
 * mode is CustomerDetailView.vue's "Add Service" (ServiceDetailView.vue
 * only ever opens this in edit mode -- `:service="service"`), which
 * computes it as the Customer's attached, not-yet-in-service devices
 * (internal/customerdevice) and disables the "Add Service" button
 * entirely when it is empty -- see that view's own reasoning. A Service
 * is always device-specific here: with exactly one eligible device it is
 * auto-selected with no picker shown (fewer clicks when there is only
 * one possible choice); with more than one, the operator must choose.
 * On submit, creating the ServiceEquipment link that ties the two
 * together is folded into this same action rather than left as a
 * separate "remember to assign equipment on the Service page" step (see
 * feedback_merge_dont_chain_workflow_steps in project memory).
 */
const props = defineProps<{ open: boolean; locationId: string; service?: Service | null; devices: Device[] }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', service: Service): void
  (event: 'updated', service: Service): void
}>()

const deviceId = ref('')

const deviceOptions = computed(() =>
  props.devices.map((device) => ({
    value: device.id,
    label: `${device.name} — ${device.manufacturer} ${device.model} (${device.serialNumber})`,
  })),
)

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
      deviceId.value = props.devices[0]?.id ?? ''
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
      // Tying the new Service to its Device is folded into this same
      // action (see this component's own doc comment) -- but if this
      // second call fails after the Service already exists (e.g. the
      // device was attached to a different service in the moment
      // between opening this dialog and submitting it), that failure is
      // deliberately not surfaced as a blocking error here: the Service
      // was created successfully and is not in a broken state, only an
      // unassigned one, exactly like any other Service before this
      // shortcut existed -- the operator can still assign equipment from
      // the Service page's own "Assign Equipment" action. There is no
      // cross-repository transaction in this codebase (see e.g.
      // internal/serviceequipment/service.ServiceEquipmentService.Create's
      // own doc comment on the same limitation), so this mirrors the
      // partial-failure handling every other composed write here already
      // accepts.
      if (deviceId.value) {
        try {
          await createServiceEquipment({
            serviceId: service.id,
            deviceId: deviceId.value,
            role: 'ONU',
            description: '',
          })
        } catch {
          // Intentionally swallowed -- see the comment above.
        }
      }
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
      <p v-else-if="!service && devices.length === 0" class="service-form__error" role="alert">
        This customer has no device available for a new service — attach one first.
      </p>
      <template v-else>
        <BaseSelect v-if="!service && devices.length > 1" v-model="deviceId" label="Device" :options="deviceOptions" />
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
          v-if="products.length > 0 && serviceProfiles.length > 0 && (service || devices.length > 0)"
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
