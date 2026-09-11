<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { createService, updateService, deleteService } from '@/services/services/serviceRepository'
import { createServiceEquipment, deleteServiceEquipment } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { runWorkflow } from '@/services/workflow/workflowRepository'
import { listProducts } from '@/services/products/productRepository'
import { listProviders } from '@/services/providers/providerRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Service } from '@/types/service'
import type { Product, ServiceType } from '@/types/product'
import type { Device } from '@/types/device'
import { UNI_PORT_OPTIONS } from '@/types/serviceEquipment'

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
 * is always device-specific here: the eligible device is always
 * auto-selected as a default, but the picker itself is only hidden when
 * `attachedDeviceCount` -- the Customer's total attached devices,
 * eligible or not -- is 1. A Customer with two attached devices, one
 * already in service, still sees the picker naming the one eligible
 * choice: with a second device on file at all, which device is about to
 * be configured should never be implicit, even though only one is
 * actually selectable right now. Only when there is truly just one
 * device attached, period, is there no ambiguity left to surface.
 * The LAN Port picker (10GE/1GE, i.e. uni 1/2 -- see
 * types/serviceEquipment.ts's UNI_PORT_OPTIONS and
 * internal/serviceequipment.ServiceEquipment's own doc comment) is the
 * other half of that same choice: exactly one of the two, never both,
 * which a single required <select> enforces for free.
 *
 * On submit, creating the ServiceEquipment link that ties the Service to
 * its Device is folded into this same action rather than left as a
 * separate "remember to assign equipment on the Service page" step (see
 * feedback_merge_dont_chain_workflow_steps in project memory) -- and so
 * is actually running the real provision-service workflow
 * (internal/workflow/engine, internal/provisioning/kontron) against the
 * ONU, rather than leaving the operator to separately open the new
 * Service and click "Provision Service" there themselves. This can take
 * a few seconds (a real SSH round trip to the OLT) and, like that same
 * button on ServiceDetailView.vue, blocks the form while it runs.
 *
 * Unlike an ordinary partial-failure composed write elsewhere in this
 * codebase, this one is all-or-nothing (2026-09-10, at the user's
 * explicit request after a real OLT-side misconfiguration created a
 * Service that could never actually serve the customer): if tying the
 * Device to the Service fails, or the real provisioning workflow fails
 * for any reason -- OLT or otherwise -- the ServiceEquipment record (if
 * it was created) and the Service itself are deleted again before this
 * function returns, and the error is shown inline in this still-open
 * dialog instead. There is no cross-repository transaction in this
 * codebase (see internal/serviceequipment/service's own doc comment on
 * the same limitation), so this is a best-effort compensating rollback,
 * not a real transaction -- if the browser tab closes mid-rollback, an
 * orphaned Service/ServiceEquipment pair can still be left behind, the
 * same way any other multi-step write here already can. This is why
 * Remove Service (see internal/service/postgres's own migration
 * 00040 comment) was also relaxed around the same time: an operator who
 * finds one of those orphans must still be able to clear it out by hand.
 *
 * Service Type (Residential/Business/Internal -- see types/product.ts)
 * is a Product field, not a Service field: there is no serviceType on
 * Service at all (see types/service.ts's own doc comment on why -- it
 * is reached by joining through Product, never duplicated). The picker
 * here is purely a UI narrowing device for the Product list, the same
 * cascading-parent-picker pattern DeviceFormDialog.vue's Manufacturer
 * picker uses to narrow Model: a computed filters productOptions by the
 * chosen Service Type, a watcher clears productId only when it stops
 * matching a user-driven Service Type change, and edit mode
 * reverse-derives the initial Service Type from the existing Service's
 * Product rather than reading a field that was never sent over the
 * wire.
 */
const props = defineProps<{
  open: boolean
  locationId: string
  service?: Service | null
  devices: Device[]
  attachedDeviceCount: number
}>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', service: Service): void
  (event: 'updated', service: Service): void
}>()

const deviceId = ref('')
const uniPort = ref('1')

const deviceOptions = computed(() =>
  props.devices.map((device) => ({
    value: device.id,
    label: `${device.name} — ${device.manufacturer} ${device.model} (${device.serialNumber})`,
  })),
)

const products = ref<Product[]>([])
const providerNameById = ref<Map<string, string>>(new Map())
const showProvider = ref(false)
const loadingOptions = ref(false)

const productId = ref('')
const serviceType = ref<ServiceType>('Residential')
const status = ref<Service['status']>('Active')
const description = ref('')
const submitting = ref(false)
const provisioning = ref(false)
const error = ref<string | null>(null)

const statusOptions = [
  { value: 'Pending', label: 'Pending' },
  { value: 'Active', label: 'Active' },
  { value: 'Suspended', label: 'Suspended' },
  { value: 'Disconnected', label: 'Disconnected' },
]

const serviceTypeOptions: { value: ServiceType; label: string }[] = [
  { value: 'Residential', label: 'Residential' },
  { value: 'Business', label: 'Business' },
  { value: 'Internal', label: 'Internal' },
]

/**
 * "<Provider name> > <Product name>", or just the Product name once
 * showProvider is false -- the exact same "only show Provider once it's
 * not the only one" rule serviceLabels.ts's resolveServiceLabels already
 * applies to how an existing Service is labeled elsewhere in this app
 * (CustomerDetailView.vue's Services table, ServiceDetailView.vue's
 * header), applied here to the picker that chooses one in the first
 * place, so the two never disagree about what a Product is called.
 *
 * Filtered to the chosen Service Type first -- the cascading-child half
 * of the Manufacturer/Model pattern this component's own doc comment
 * describes (see DeviceFormDialog.vue's modelOptions).
 */
const productOptions = computed(() =>
  products.value
    .filter((product) => product.serviceType === serviceType.value)
    .map((product) => ({
      value: product.id,
      label:
        showProvider.value && providerNameById.value.has(product.providerId)
          ? `${providerNameById.value.get(product.providerId)} > ${product.name}`
          : product.name,
    })),
)

// Switching Service Type clears Product whenever it no longer belongs
// to the newly-selected Service Type -- mirrors DeviceFormDialog.vue's
// manufacturerId watcher exactly, including why it never fights the
// fetch-then-populate watcher below (see that watcher's own comment).
watch(serviceType, () => {
  if (!products.value.some((p) => p.id === productId.value && p.serviceType === serviceType.value)) {
    productId.value = ''
  }
})

// Products and Providers are fetched fresh each time the dialog opens
// rather than once at app startup -- there is no Product/Provider
// Workspace to keep a cached copy fresh against, and this dataset is
// small enough that refetching is simpler than inventing a cache to
// invalidate. Needed in both modes: create defaults to the first
// option, edit needs the options list to show the service's current
// selection.
watch(
  () => props.open,
  async (isOpen) => {
    if (!isOpen) return
    error.value = null
    loadingOptions.value = true
    const [productList, providerList] = await Promise.all([listProducts(), listProviders()])
    products.value = productList
    providerNameById.value = new Map(providerList.map((provider) => [provider.id, provider.name]))
    showProvider.value = providerList.length > 1

    if (props.service) {
      productId.value = props.service.productId
      serviceType.value = productList.find((p) => p.id === props.service?.productId)?.serviceType ?? 'Residential'
      status.value = props.service.status
      description.value = props.service.description
    } else {
      serviceType.value = 'Residential'
      productId.value = productList.find((p) => p.serviceType === serviceType.value)?.id ?? ''
      status.value = 'Active'
      description.value = ''
      deviceId.value = props.devices[0]?.id ?? ''
      uniPort.value = '1'
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
        status: status.value,
        description: description.value,
      })

      // Tying the new Service to its Device, and actually provisioning
      // it, are folded into this same action (see this component's own
      // doc comment) -- and, since 2026-09-10, this whole action is
      // all-or-nothing: any failure here, tying the Device or
      // provisioning it, rolls back the ServiceEquipment (if created)
      // and the Service itself rather than leaving a Service behind that
      // was never actually applied to the device.
      let equipmentId: string | null = null
      try {
        if (!deviceId.value) {
          throw new Error('A device is required to add a service.')
        }
        const equipment = await createServiceEquipment({
          serviceId: service.id,
          deviceId: deviceId.value,
          role: 'ONU',
          uniPort: Number(uniPort.value),
        })
        equipmentId = equipment.id

        provisioning.value = true
        // runWorkflow only throws if the instance never reaches a
        // terminal status within its polling window (see that
        // function's own doc comment) -- a clean Failed/Cancelled result
        // resolves normally, so it has to be checked explicitly here
        // rather than assumed to always throw on failure.
        const instance = await runWorkflow(service.id, 'provision-service')
        if (instance.status !== 'Succeeded') {
          throw new Error(instance.errorMessage ?? 'The service could not be provisioned onto the device.')
        }
      } catch (createErr) {
        // Best-effort, independent deletes -- see this component's own
        // doc comment on why this is a compensating rollback, not a real
        // transaction. deleteServiceEquipment runs first: services.id is
        // still referenced by service_equipment.service_id (ON DELETE
        // RESTRICT), so deleting the Service first would just fail.
        if (equipmentId) {
          try {
            await deleteServiceEquipment(equipmentId)
          } catch {
            // Best-effort -- see above.
          }
        }
        try {
          await deleteService(service.id)
        } catch {
          // Best-effort -- see above.
        }
        error.value =
          createErr instanceof ApiError || createErr instanceof Error
            ? createErr.message
            : 'The service could not be added.'
        return
      } finally {
        provisioning.value = false
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
    <p v-if="loadingOptions" class="service-form__loading">Loading products…</p>

    <form v-else class="service-form" @submit.prevent="handleSubmit">
      <p v-if="products.length === 0" class="service-form__error" role="alert">
        No products exist yet — create one in the catalog before adding a service.
      </p>
      <p v-else-if="!service && devices.length === 0" class="service-form__error" role="alert">
        This customer has no device available for a new service — attach one first.
      </p>
      <template v-else>
        <BaseSelect
          v-if="!service && attachedDeviceCount > 1"
          v-model="deviceId"
          label="Device"
          :options="deviceOptions"
        />
        <BaseSelect v-if="!service" v-model="uniPort" label="LAN Port" :options="UNI_PORT_OPTIONS" />
        <BaseSelect v-model="serviceType" label="Service Type" :options="serviceTypeOptions" />
        <BaseSelect v-model="productId" label="Product" :options="productOptions" />
        <BaseSelect v-if="service" v-model="status" label="Status" :options="statusOptions" />
        <BaseInput v-model="description" label="Description" />
      </template>

      <p v-if="error" class="service-form__error" role="alert">{{ error }}</p>

      <div class="service-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton
          v-if="products.length > 0 && (service || devices.length > 0)"
          type="submit"
          variant="primary"
          :disabled="submitting"
        >
          {{ provisioning ? 'Provisioning…' : submitting ? 'Saving…' : service ? 'Save Changes' : 'Add Service' }}
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
