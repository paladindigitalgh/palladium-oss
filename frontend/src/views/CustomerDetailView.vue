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
import ContactFormDialog from '@/components/dialogs/ContactFormDialog.vue'
import CustomerFormDialog from '@/components/dialogs/CustomerFormDialog.vue'
import LocationFormDialog from '@/components/dialogs/LocationFormDialog.vue'
import ServiceFormDialog from '@/components/dialogs/ServiceFormDialog.vue'
import { getCustomerById, deleteCustomer } from '@/services/customers/customerRepository'
import { listContactsByCustomerId, deleteContact } from '@/services/contacts/contactRepository'
import { listLocationsByCustomerId, deleteLocation } from '@/services/locations/locationRepository'
import { listServicesByLocationIds, deleteService } from '@/services/services/serviceRepository'
import { resolveServiceLabels } from '@/services/services/serviceLabels'
import { listEvents } from '@/services/events/eventRepository'
import {
  listCustomerEquipmentLocations,
  runONURunningConfig,
  runONUStatus,
  runONUEthernetPorts,
  runDHCPSnoopingEntries,
  runMACAddressTableEntries,
} from '@/services/diagnostics/diagnosticsRepository'
import { getOLTById } from '@/services/olts/oltRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Contact } from '@/types/contact'
import type { Customer } from '@/types/customer'
import type { Location } from '@/types/location'
import type { Service } from '@/types/service'
import type { TimelineEvent } from '@/types/timelineEvent'
import type { CustomerEquipmentLocation } from '@/types/onuDiagnostics'
import type { OLT } from '@/types/olt'

/**
 * The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace"), backed by the real backend.
 *
 * Sections that depended on concepts the backend does not model at all
 * (Alerts) are removed rather than faked. Contacts, Locations, and
 * Services are all real, resolved on demand (docs/03-DOMAIN-MODEL.md: a
 * Customer owns Services through Locations, and equipment is associated
 * through Services -- never embedded on Customer itself; Contacts are
 * the same shape one level simpler, with no further child of their own).
 * Timeline is real Events (docs/02-DESIGN-PRINCIPLES.md principle 10).
 *
 * Create/edit/delete lets an operator build up (and tear down) a test
 * customer the same way a real onboarding would: customer, then contact,
 * then location, then service. Deletes go through the backend's real
 * foreign key restrictions (customers <- locations <- services) rather
 * than cascading -- a blocked delete surfaces a specific, friendly
 * message instead of the raw backend error. Contacts are the one
 * exception: contacts.customer_id is ON DELETE CASCADE, not RESTRICT
 * (see internal/contact/postgres/contact.go's doc comment), so removing
 * a Contact never blocks anything and deleting the Customer itself
 * removes its Contacts along with it.
 */
const route = useRoute()
const router = useRouter()

const customer = ref<Customer | null>(null)
const contacts = ref<Contact[]>([])
const locations = ref<Location[]>([])
const services = ref<Service[]>([])
const serviceLabelsById = ref<Map<string, string>>(new Map())
const timeline = ref<TimelineEvent[]>([])
const equipmentLocations = ref<CustomerEquipmentLocation[]>([])
const oltsById = ref<Map<string, OLT>>(new Map())
const onuDiagnostics = ref<Map<string, ONUDiagnosticsState>>(new Map())
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  customer.value = null
  contacts.value = []
  locations.value = []
  services.value = []
  serviceLabelsById.value = new Map()
  timeline.value = []
  equipmentLocations.value = []
  oltsById.value = new Map()
  onuDiagnostics.value = new Map()

  const result = await getCustomerById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  customer.value = result

  const [customerContacts, customerLocations, events, customerEquipmentLocations] = await Promise.all([
    listContactsByCustomerId(id),
    listLocationsByCustomerId(id),
    listEvents('customer', id),
    listCustomerEquipmentLocations(id),
  ])
  contacts.value = customerContacts
  locations.value = customerLocations
  timeline.value = events
  services.value = await listServicesByLocationIds(customerLocations.map((location) => location.id))
  serviceLabelsById.value = await resolveServiceLabels(services.value)

  equipmentLocations.value = customerEquipmentLocations
  const uniqueOltIds = [...new Set(customerEquipmentLocations.map((item) => item.oltId))]
  const olts = await Promise.all(uniqueOltIds.map((oltId) => getOLTById(oltId)))
  const byOltId = new Map<string, OLT>()
  uniqueOltIds.forEach((oltId, index) => {
    const olt = olts[index]
    if (olt) byOltId.set(oltId, olt)
  })
  oltsById.value = byOltId

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

const contactColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Name' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

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

// --- Add/Edit/Remove Contact ---

const showContactForm = ref(false)

function handleContactCreated(contact: Contact) {
  showContactForm.value = false
  contacts.value = [...contacts.value, contact]
}

const contactEditTarget = ref<Contact | null>(null)

function handleContactUpdated(updated: Contact) {
  contacts.value = contacts.value.map((contact) => (contact.id === updated.id ? updated : contact))
  contactEditTarget.value = null
}

// No conflict branch: contacts.customer_id is ON DELETE CASCADE, and
// nothing else references a Contact, so deleteContact never throws an
// ApiError with kind "conflict" (see contactRepository.ts's own doc
// comment on deleteContact).
const contactDeleteTarget = ref<Contact | null>(null)
const contactDeletePending = ref(false)
const contactDeleteError = ref<string | null>(null)

async function confirmDeleteContact() {
  const target = contactDeleteTarget.value
  if (!target) return
  contactDeletePending.value = true
  contactDeleteError.value = null
  try {
    await deleteContact(target.id)
    contacts.value = contacts.value.filter((contact) => contact.id !== target.id)
    contactDeleteTarget.value = null
  } catch {
    contactDeleteError.value = 'The contact could not be removed.'
  } finally {
    contactDeletePending.value = false
  }
}

// --- Add/Edit/Remove Location ---

const showLocationForm = ref(false)

function handleLocationCreated(location: Location) {
  showLocationForm.value = false
  locations.value = [...locations.value, location]
}

const locationEditTarget = ref<Location | null>(null)

function handleLocationUpdated(updated: Location) {
  locations.value = locations.value.map((location) => (location.id === updated.id ? updated : location))
  locationEditTarget.value = null
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

async function handleServiceCreated(service: Service) {
  showServiceForm.value = false
  services.value = [...services.value, service]
  const labels = await resolveServiceLabels([service])
  serviceLabelsById.value = new Map(serviceLabelsById.value).set(service.id, labels.get(service.id) ?? service.id)
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

// --- ONU Diagnostics ---

interface DiagnosticCommandResult {
  label: string
  output: string | null
  error: string | null
}

interface ONUDiagnosticsState {
  pending: boolean
  results: DiagnosticCommandResult[] | null
}

/**
 * The four commands "Check ONU Status" runs, in order, against every
 * currently-attached equipment location -- each is its own SSH
 * connection to the OLT (internal/olt/connect), run sequentially rather
 * than in parallel to stay gentle on a device's own small concurrent-
 * session budget (see internal/platform/ssh's "Interactive shell mode"
 * doc comment on the real Kontron/Iskratel C16 this was confirmed
 * against). A failure on one command does not stop the rest: each is an
 * independent read, so the operator sees whatever is available even if
 * one specific query fails.
 */
const ONU_STATUS_COMMANDS: { label: string; run: (oltId: string, iface: string) => Promise<string> }[] = [
  { label: 'Running Configuration', run: runONURunningConfig },
  { label: 'Status', run: runONUStatus },
  { label: 'Ethernet Ports', run: runONUEthernetPorts },
  { label: 'DHCP Snooping', run: runDHCPSnoopingEntries },
  { label: 'MAC Address Table', run: runMACAddressTableEntries },
]

async function checkONUStatus(equipmentLocation: CustomerEquipmentLocation) {
  onuDiagnostics.value.set(equipmentLocation.serviceEquipmentId, { pending: true, results: null })

  const results: DiagnosticCommandResult[] = []
  for (const command of ONU_STATUS_COMMANDS) {
    try {
      const output = await command.run(equipmentLocation.oltId, equipmentLocation.interface)
      results.push({ label: command.label, output, error: null })
    } catch (err) {
      results.push({
        label: command.label,
        output: null,
        error: err instanceof ApiError ? err.message : 'This command failed to run.',
      })
    }
  }

  onuDiagnostics.value.set(equipmentLocation.serviceEquipmentId, { pending: false, results })
}
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

    <SectionCard title="Contacts" icon="customers" :badge="contacts.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showContactForm = true">Add Contact</BaseButton>
      </div>

      <ContactFormDialog
        :open="showContactForm"
        :customer-id="customer.id"
        @close="showContactForm = false"
        @created="handleContactCreated"
      />

      <ContactFormDialog
        :open="contactEditTarget !== null"
        :customer-id="customer.id"
        :contact="contactEditTarget"
        @close="contactEditTarget = null"
        @updated="handleContactUpdated"
      />

      <ConfirmationDialog
        :open="contactDeleteTarget !== null"
        title="Remove Contact"
        :description="`Remove ${contactDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove Contact"
        destructive
        :pending="contactDeletePending"
        :error="contactDeleteError"
        @confirm="confirmDeleteContact"
        @cancel="contactDeleteTarget = null"
      />

      <SimpleTable
        :columns="contactColumns"
        :rows="contacts"
        :row-key="(contact) => contact.id"
        empty-icon="customers"
        empty-title="No contacts on file"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-role="{ row }">{{ row.role }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click="contactEditTarget = row">Edit</BaseButton>
          <BaseButton variant="ghost" size="sm" @click="contactDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
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

      <LocationFormDialog
        :open="locationEditTarget !== null"
        :customer-id="customer.id"
        :location="locationEditTarget"
        @close="locationEditTarget = null"
        @updated="handleLocationUpdated"
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
          <BaseButton variant="ghost" size="sm" @click="locationEditTarget = row">Edit</BaseButton>
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
        :description="`Remove ${serviceDeleteTarget ? serviceLabelsById.get(serviceDeleteTarget.id) ?? serviceDeleteTarget.id : ''}? This cannot be undone.`"
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
        <template #cell-service="{ row }">{{ serviceLabelsById.get(row.id) ?? row.id }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="serviceDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="ONU Diagnostics" icon="devices" :badge="equipmentLocations.length">
      <p v-if="equipmentLocations.length === 0" class="no-relationship">
        No ONU is currently attached to this customer.
      </p>

      <div
        v-for="equipmentLocation in equipmentLocations"
        :key="equipmentLocation.serviceEquipmentId"
        class="onu-diagnostics-block"
      >
        <div class="onu-diagnostics-block__header">
          <div>
            <span class="cell-strong">{{ equipmentLocation.interface }}</span>
            <span class="onu-diagnostics-block__olt">
              on {{ oltsById.get(equipmentLocation.oltId)?.name ?? equipmentLocation.oltId }}
            </span>
          </div>
          <BaseButton
            variant="secondary"
            size="sm"
            :disabled="onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.pending"
            :disabled-reason="
              onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.pending ? 'Running…' : undefined
            "
            @click="checkONUStatus(equipmentLocation)"
          >
            {{ onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.pending ? 'Checking…' : 'Check ONU Status' }}
          </BaseButton>
        </div>

        <div v-if="onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.results" class="onu-diagnostics-results">
          <div
            v-for="result in onuDiagnostics.get(equipmentLocation.serviceEquipmentId)!.results"
            :key="result.label"
            class="onu-diagnostics-result"
          >
            <h4 class="onu-diagnostics-result__label">{{ result.label }}</h4>
            <p v-if="result.error" class="onu-diagnostics-result__error" role="alert">{{ result.error }}</p>
            <pre v-else class="onu-diagnostics-result__output">{{ result.output }}</pre>
          </div>
        </div>
      </div>
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

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.onu-diagnostics-block {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
}

.onu-diagnostics-block + .onu-diagnostics-block {
  margin-top: var(--space-4);
}

.onu-diagnostics-block__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.onu-diagnostics-block__olt {
  margin-left: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.onu-diagnostics-results {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-top: var(--space-4);
}

.onu-diagnostics-result__label {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.onu-diagnostics-result__output {
  margin: 0;
  padding: var(--space-3);
  background-color: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
  color: var(--color-text-primary);
}

.onu-diagnostics-result__error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
