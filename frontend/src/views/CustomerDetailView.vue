<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import CustomerFormDialog from '@/components/dialogs/CustomerFormDialog.vue'
import LocationFormDialog from '@/components/dialogs/LocationFormDialog.vue'
import ServiceFormDialog from '@/components/dialogs/ServiceFormDialog.vue'
import { getCustomerById, deleteCustomer } from '@/services/customers/customerRepository'
import { listLocationsByCustomerId, deleteLocation } from '@/services/locations/locationRepository'
import { listServicesByLocationIds, deleteService } from '@/services/services/serviceRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Customer } from '@/types/customer'
import type { Location } from '@/types/location'
import type { Service } from '@/types/service'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace"), backed by the real backend.
 *
 * Sections that depended on concepts the backend does not model at all
 * (Contacts, Alerts) are removed rather than faked. Locations and
 * Services are real, resolved on demand (docs/03-DOMAIN-MODEL.md: a
 * Customer owns Services through Locations, and equipment is associated
 * through Services -- never embedded on Customer itself). Timeline is
 * real Events (docs/02-DESIGN-PRINCIPLES.md principle 10).
 *
 * Create/edit/delete lets an operator build up (and tear down) a test
 * customer the same way a real onboarding would: customer, then
 * location, then service. Deletes go through the backend's real foreign
 * key restrictions (customers <- locations <- services) rather than
 * cascading -- a blocked delete surfaces a specific, friendly message
 * instead of the raw backend error.
 */
const route = useRoute()
const router = useRouter()

const customer = ref<Customer | null>(null)
const locations = ref<Location[]>([])
const services = ref<Service[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  customer.value = null
  locations.value = []
  services.value = []
  timeline.value = []

  const result = await getCustomerById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  customer.value = result

  const [customerLocations, events] = await Promise.all([listLocationsByCustomerId(id), listEvents('customer', id)])
  locations.value = customerLocations
  timeline.value = events
  services.value = await listServicesByLocationIds(customerLocations.map((location) => location.id))

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const c = customer.value
  if (!c) return []
  return [
    { icon: 'health', label: 'Status', value: c.status },
    { icon: 'customers', label: 'Customer Type', value: c.customerType },
    { icon: 'clock', label: 'Created', value: formatDate(c.createdAt) },
  ]
})

const locationColumns: SimpleTableColumn[] = [
  { key: 'location', label: 'Location' },
  { key: 'type', label: 'Type' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

const serviceColumns: SimpleTableColumn[] = [
  { key: 'service', label: 'Service' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

function serviceRowKey(service: Service): string {
  return service.id
}

function openService(service: Service) {
  router.push(`/services/${service.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Edit Customer ---

const showEditCustomerDialog = ref(false)

function handleCustomerUpdated(updated: Customer) {
  customer.value = updated
  showEditCustomerDialog.value = false
}

// --- Delete Customer ---

const showDeleteCustomerDialog = ref(false)
const deleteCustomerPending = ref(false)
const deleteCustomerError = ref<string | null>(null)

async function confirmDeleteCustomer() {
  if (!customer.value) return
  deleteCustomerPending.value = true
  deleteCustomerError.value = null
  try {
    await deleteCustomer(customer.value.id)
    router.push('/customers')
  } catch (err) {
    deleteCustomerError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This customer still has locations attached — remove those first.'
        : 'The customer could not be deleted.'
  } finally {
    deleteCustomerPending.value = false
  }
}

// --- Add/Remove Location ---

const showLocationForm = ref(false)

function handleLocationCreated(location: Location) {
  showLocationForm.value = false
  locations.value = [...locations.value, location]
}

const locationDeleteTarget = ref<Location | null>(null)
const locationDeletePending = ref(false)
const locationDeleteError = ref<string | null>(null)

async function confirmDeleteLocation() {
  const target = locationDeleteTarget.value
  if (!target) return
  locationDeletePending.value = true
  locationDeleteError.value = null
  try {
    await deleteLocation(target.id)
    locations.value = locations.value.filter((location) => location.id !== target.id)
    locationDeleteTarget.value = null
  } catch (err) {
    locationDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This location still has services attached — remove those first.'
        : 'The location could not be deleted.'
  } finally {
    locationDeletePending.value = false
  }
}

// --- Add/Remove Service ---

const showServiceForm = ref(false)
const serviceFormLocationId = ref('')

function openServiceForm() {
  serviceFormLocationId.value = locations.value[0]?.id ?? ''
  showServiceForm.value = true
}

function handleServiceCreated(service: Service) {
  showServiceForm.value = false
  services.value = [...services.value, service]
}

const serviceDeleteTarget = ref<Service | null>(null)
const serviceDeletePending = ref(false)
const serviceDeleteError = ref<string | null>(null)

async function confirmDeleteService() {
  const target = serviceDeleteTarget.value
  if (!target) return
  serviceDeletePending.value = true
  serviceDeleteError.value = null
  try {
    await deleteService(target.id)
    services.value = services.value.filter((service) => service.id !== target.id)
    serviceDeleteTarget.value = null
  } catch (err) {
    serviceDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This service still has equipment or workflow history — remove those first.'
        : 'The service could not be deleted.'
  } finally {
    serviceDeletePending.value = false
  }
}

const locationOptions = computed(() => locations.value.map((location) => ({ value: location.id, label: location.name })))
</script>

<template>
  <div v-if="loading" class="customer-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="customer-detail-view__status">
    <BaseErrorState
      title="Customer not found"
      description="This customer may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/customers')">Back to Customers</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="customer">
    <WorkspaceHeader
      :title="customer.name"
      :subtitle="`${customer.customerType} Customer`"
      :status="{ label: customer.status, variant: customer.status === 'Active' ? 'success' : 'neutral' }"
      :metadata="[`Customer ${customer.id}`]"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditCustomerDialog = true">Edit Customer</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteCustomerDialog = true">
              Delete Customer
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <CustomerFormDialog
      :open="showEditCustomerDialog"
      :customer="customer"
      @close="showEditCustomerDialog = false"
      @updated="handleCustomerUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteCustomerDialog"
      title="Delete Customer"
      :description="`Permanently delete ${customer.name}? This cannot be undone.`"
      confirm-label="Delete Customer"
      destructive
      :pending="deleteCustomerPending"
      :error="deleteCustomerError"
      @confirm="confirmDeleteCustomer"
      @cancel="showDeleteCustomerDialog = false"
    />

    <SectionCard title="Summary" icon="customers">
      <FactGrid :facts="summaryFacts" />
      <p v-if="customer.description" class="customer-description">{{ customer.description }}</p>
    </SectionCard>

    <SectionCard title="Locations" icon="location" :badge="locations.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showLocationForm = true">Add Location</BaseButton>
      </div>

      <LocationFormDialog
        :open="showLocationForm"
        :customer-id="customer.id"
        @close="showLocationForm = false"
        @created="handleLocationCreated"
      />

      <ConfirmationDialog
        :open="locationDeleteTarget !== null"
        title="Remove Location"
        :description="`Remove ${locationDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove Location"
        destructive
        :pending="locationDeletePending"
        :error="locationDeleteError"
        @confirm="confirmDeleteLocation"
        @cancel="locationDeleteTarget = null"
      />

      <SimpleTable
        :columns="locationColumns"
        :rows="locations"
        :row-key="(location) => location.id"
        empty-icon="location"
        empty-title="No locations on file"
      >
        <template #cell-location="{ row }">
          <div class="location-cell">
            <span class="cell-strong">{{ row.name }}</span>
            <span class="location-cell__address">{{ row.address1 }}, {{ row.city }}, {{ row.state }} {{ row.postalCode }}</span>
          </div>
        </template>
        <template #cell-type="{ row }">{{ row.type }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click="locationDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Services" icon="services" :badge="services.length">
      <div class="section-toolbar">
        <BaseSelect
          v-if="locations.length > 1"
          v-model="serviceFormLocationId"
          label="Location"
          :options="locationOptions"
        />
        <BaseButton
          variant="secondary"
          size="sm"
          :disabled="locations.length === 0"
          :disabled-reason="locations.length === 0 ? 'Add a location first' : undefined"
          @click="openServiceForm"
        >
          Add Service
        </BaseButton>
      </div>

      <ServiceFormDialog
        :open="showServiceForm"
        :location-id="serviceFormLocationId"
        @close="showServiceForm = false"
        @created="handleServiceCreated"
      />

      <ConfirmationDialog
        :open="serviceDeleteTarget !== null"
        title="Remove Service"
        :description="`Remove service ${serviceDeleteTarget?.id}? This cannot be undone.`"
        confirm-label="Remove Service"
        destructive
        :pending="serviceDeletePending"
        :error="serviceDeleteError"
        @confirm="confirmDeleteService"
        @cancel="serviceDeleteTarget = null"
      />

      <SimpleTable
        :columns="serviceColumns"
        :rows="services"
        :row-key="serviceRowKey"
        clickable
        empty-icon="services"
        empty-title="No services on this account"
        @row-click="openService"
      >
        <template #cell-service="{ row }">
          <span class="cell-mono">{{ row.id }}</span>
        </template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="serviceDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.customer-detail-view__status {
  padding: var(--space-6);
}

.cell-strong {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.cell-mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.customer-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.location-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.location-cell__address {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.section-toolbar {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
</style>
