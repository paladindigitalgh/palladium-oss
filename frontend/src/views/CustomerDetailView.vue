<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import NotesSection from '@/components/data-display/NotesSection.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import ContactFormDialog from '@/components/dialogs/ContactFormDialog.vue'
import CustomerFormDialog from '@/components/dialogs/CustomerFormDialog.vue'
import LocationFormDialog from '@/components/dialogs/LocationFormDialog.vue'
import ServiceFormDialog from '@/components/dialogs/ServiceFormDialog.vue'
import AttachCustomerDeviceDialog from '@/components/dialogs/AttachCustomerDeviceDialog.vue'
import RemoveCustomerDialog from '@/components/dialogs/RemoveCustomerDialog.vue'
import { getCustomerById } from '@/services/customers/customerRepository'
import { listContactsByCustomerId, deleteContact } from '@/services/contacts/contactRepository'
import { listLocationsByCustomerId, deleteLocation } from '@/services/locations/locationRepository'
import { listServicesByLocationIds, deleteService } from '@/services/services/serviceRepository'
import { runWorkflow } from '@/services/workflow/workflowRepository'
import { resolveServiceLabels } from '@/services/services/serviceLabels'
import { listEvents } from '@/services/events/eventRepository'
import { listNotes, createNote } from '@/services/notes/noteRepository'
import {
  listCustomerEquipmentLocations,
  runONURunningConfig,
  runONUStatus,
  runONUEthernetPorts,
  runDHCPSnoopingEntries,
  runMACAddressTableEntries,
} from '@/services/diagnostics/diagnosticsRepository'
import { getOLTById } from '@/services/olts/oltRepository'
import { getDeviceById } from '@/services/devices/deviceRepository'
import {
  listActiveCustomerDevicesByCustomerId,
  detachCustomerDevice,
  setCustomerDeviceLocation,
} from '@/services/customerDevices/customerDeviceRepository'
import { listServiceEquipment, listServiceEquipmentByServiceId, deleteServiceEquipment } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Contact } from '@/types/contact'
import type { Customer } from '@/types/customer'
import type { Location } from '@/types/location'
import type { Service } from '@/types/service'
import type { TimelineEvent } from '@/types/timelineEvent'
import type { Note } from '@/types/note'
import type { CustomerEquipmentLocation } from '@/types/onuDiagnostics'
import type { OLT } from '@/types/olt'
import type { ServiceEquipment } from '@/types/serviceEquipment'
import type { Device } from '@/types/device'
import type { CustomerDevice } from '@/types/customerDevice'

/**
 * The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace"), backed by the real backend.
 *
 * Sections that depended on concepts the backend does not model at all
 * (Alerts) are removed rather than faked. Contacts, Locations, Devices,
 * and Services are all real, resolved on demand (docs/03-DOMAIN-MODEL.md:
 * a Customer owns Services through Locations; Contacts are the same
 * shape one level simpler, with no further child of their own). Devices
 * are the one deliberate exception to "equipment is associated through
 * Services, never embedded on Customer itself": internal/customerdevice
 * lets a Device be attached to a Customer directly, independent of any
 * Service, for the real install-before-activation case (see that
 * package's own doc comment). Timeline is real Events
 * (docs/02-DESIGN-PRINCIPLES.md principle 10).
 *
 * Create/edit/delete lets an operator build up (and tear down) a test
 * customer the same way a real onboarding would: customer, then contact,
 * then location, then device, then service -- Add Service is disabled
 * until at least one attached Device is eligible (Unused, not already
 * serving another Service), and creating a Service ties it to that
 * Device in the same action (see ServiceFormDialog.vue's own doc
 * comment). Deletes go through the backend's real foreign key
 * restrictions (customers <- locations <- services) rather than
 * cascading -- a blocked delete surfaces a specific, friendly message
 * instead of the raw backend error. Contacts are the one exception:
 * contacts.customer_id is ON DELETE CASCADE, not RESTRICT (see
 * internal/contact/postgres/contact.go's doc comment), so removing a
 * Contact never blocks anything and deleting the Customer itself removes
 * its Contacts along with it.
 */
const route = useRoute()
const router = useRouter()

const customer = ref<Customer | null>(null)
const contacts = ref<Contact[]>([])
const locations = ref<Location[]>([])
const customerDevices = ref<CustomerDevice[]>([])
const devicesById = ref<Map<string, Device>>(new Map())
const activeServiceEquipment = ref<ServiceEquipment[]>([])
const services = ref<Service[]>([])
const serviceLabelsById = ref<Map<string, string>>(new Map())
const timeline = ref<TimelineEvent[]>([])
const notes = ref<Note[]>([])
const notesSubmitting = ref(false)
const notesError = ref<string | null>(null)
const equipmentLocations = ref<CustomerEquipmentLocation[]>([])
const oltsById = ref<Map<string, OLT>>(new Map())
const onuDiagnostics = ref<Map<string, ONUDiagnosticsState>>(new Map())
const loading = ref(true)
const notFound = ref(false)

/**
 * Re-resolves the full Device record for every currently-attached
 * CustomerDevice, refreshing devicesById -- purely for display (name,
 * manufacturer/model, serial, Status) in the Devices section and the
 * Device picker's option labels. Called on initial load and again after
 * attaching a Device.
 */
async function refreshAttachedDevices() {
  const attachedDevices = await Promise.all(customerDevices.value.map((cd) => getDeviceById(cd.deviceId)))
  const byDeviceId = new Map<string, Device>()
  customerDevices.value.forEach((cd, index) => {
    const device = attachedDevices[index]
    if (device) byDeviceId.set(cd.deviceId, device)
  })
  devicesById.value = byDeviceId
}

/**
 * Re-fetches every active serviceequipment.ServiceEquipment record
 * system-wide into activeServiceEquipment -- the one source three
 * separate computed lookups below derive from, each keyed differently
 * for its own caller:
 *  - activeServiceDeviceIds -- which Devices currently fulfill one --
 *    what eligibleServiceDevices below actually gates on. Deliberately
 *    not derived from Device.Status: since
 *    internal/customerdevice/service.CustomerDeviceService started
 *    marking a Device Active purely from being attached to a Customer
 *    (see project memory on that fix), Status alone can no longer
 *    answer "does this Device already have a Service" -- a Device
 *    attached here and nothing else is Active too, and should still be
 *    offered.
 *  - serviceDeviceIdById -- which one Device each Service is currently
 *    tied to, keyed by Service ID -- what the Services section's Device
 *    column reads, so an operator with more than one Service can tell
 *    them apart without opening each one.
 *  - deviceIdByServiceEquipmentId -- the same relationship keyed by
 *    ServiceEquipment ID instead, since that is what
 *    CustomerEquipmentLocation.serviceEquipmentId (ONU Diagnostics
 *    below) addresses, not a Service ID.
 * Called on initial load and again after creating or removing a
 * Service, the two actions that can change any of the three.
 */
async function refreshActiveServiceEquipment() {
  activeServiceEquipment.value = (await listServiceEquipment()).filter((e) => e.removedAt === null)
}

async function refreshNotes() {
  if (!customer.value) return
  notes.value = await listNotes('customer', customer.value.id)
}

async function handleAddNote(body: string) {
  if (!customer.value) return
  notesSubmitting.value = true
  notesError.value = null
  try {
    await createNote({ entityType: 'customer', entityId: customer.value.id, body })
    await refreshNotes()
  } catch {
    notesError.value = 'The note could not be added.'
  } finally {
    notesSubmitting.value = false
  }
}

const activeServiceDeviceIds = computed(() => new Set(activeServiceEquipment.value.map((e) => e.deviceId)))
const serviceDeviceIdById = computed(() => new Map(activeServiceEquipment.value.map((e) => [e.serviceId, e.deviceId])))
const deviceIdByServiceEquipmentId = computed(() => new Map(activeServiceEquipment.value.map((e) => [e.id, e.deviceId])))

async function load(id: string) {
  loading.value = true
  notFound.value = false
  customer.value = null
  contacts.value = []
  locations.value = []
  customerDevices.value = []
  devicesById.value = new Map()
  activeServiceEquipment.value = []
  services.value = []
  serviceLabelsById.value = new Map()
  timeline.value = []
  notes.value = []
  equipmentLocations.value = []
  oltsById.value = new Map()
  onuDiagnostics.value = new Map()
  collapsedONUDiagnostics.value = new Set()

  const result = await getCustomerById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  customer.value = result

  const [customerContacts, customerLocations, events, customerNotes, customerEquipmentLocations, activeCustomerDevices] =
    await Promise.all([
      listContactsByCustomerId(id),
      listLocationsByCustomerId(id),
      listEvents('customer', id),
      listNotes('customer', id),
      listCustomerEquipmentLocations(id),
      listActiveCustomerDevicesByCustomerId(id),
      refreshActiveServiceEquipment(),
    ])
  contacts.value = customerContacts
  notes.value = customerNotes
  locations.value = customerLocations
  timeline.value = events
  services.value = await listServicesByLocationIds(customerLocations.map((location) => location.id))
  serviceLabelsById.value = await resolveServiceLabels(services.value)

  customerDevices.value = activeCustomerDevices
  await refreshAttachedDevices()

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
  { key: 'device', label: 'Device' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

const deviceColumns: SimpleTableColumn[] = [
  { key: 'device', label: 'Device' },
  { key: 'location', label: 'Location' },
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

// --- Remove Customer ---

const showRemoveCustomerDialog = ref(false)

function handleCustomerRemoved() {
  showRemoveCustomerDialog.value = false
  router.push('/customers')
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
    // The backend now names the specific relationship that's still
    // attached (a Service, or a Device's placement history) rather than
    // this always guessing "services" -- see
    // internal/location/postgres/errors.go's fkViolationReasons.
    locationDeleteError.value = err instanceof ApiError ? err.message : 'The location could not be deleted.'
  } finally {
    locationDeletePending.value = false
  }
}

// --- Attach/Detach Device ---

const showAttachDeviceDialog = ref(false)

async function handleDeviceAttached(record: CustomerDevice) {
  showAttachDeviceDialog.value = false
  customerDevices.value = [...customerDevices.value, record]
  const device = await getDeviceById(record.deviceId)
  if (device) devicesById.value = new Map(devicesById.value).set(device.id, device)
}

function openDevice(record: CustomerDevice) {
  router.push(`/devices/${record.deviceId}`)
}

const deviceDetachTarget = ref<CustomerDevice | null>(null)
const deviceDetachPending = ref(false)
const deviceDetachError = ref<string | null>(null)

async function confirmDetachDevice() {
  const target = deviceDetachTarget.value
  if (!target) return
  deviceDetachPending.value = true
  deviceDetachError.value = null
  try {
    await detachCustomerDevice(target)
    customerDevices.value = customerDevices.value.filter((cd) => cd.id !== target.id)
    deviceDetachTarget.value = null
  } catch (err) {
    deviceDetachError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This device still fulfills an active service — remove it from the service before detaching.'
        : 'The device could not be detached.'
  } finally {
    deviceDetachPending.value = false
  }
}

const deviceLocationError = ref<string | null>(null)
const deviceLocationPending = ref<string | null>(null)

/**
 * Changes which of the Customer's own Locations an already-attached
 * Device is recorded as sitting at -- purely operator tracking (see
 * CustomerDevice.locationId's own doc comment), correcting or filling in
 * that field without detaching and reattaching the Device.
 */
async function handleDeviceLocationChanged(record: CustomerDevice, locationId: string) {
  deviceLocationError.value = null
  deviceLocationPending.value = record.id
  try {
    const updated = await setCustomerDeviceLocation(record, locationId === '' ? null : locationId)
    customerDevices.value = customerDevices.value.map((cd) => (cd.id === updated.id ? updated : cd))
  } catch {
    deviceLocationError.value = 'The location could not be updated for this device.'
  } finally {
    deviceLocationPending.value = null
  }
}

/**
 * The devices ServiceFormDialog's create mode may pick from: attached to
 * this customer, and not already fulfilling another active Service (the
 * same uniqueness rule internal/serviceequipment/service enforces
 * server-side -- see activeServiceDeviceIds/refreshActiveServiceEquipment
 * above for why this is not simply "Device.Status === 'Unused'": Status
 * is Active for any attached Device now, service or no service, so it
 * can no longer answer this question by itself). This is also what "Add
 * Service" gates on below.
 */
const eligibleServiceDevices = computed(() =>
  customerDevices.value
    .filter((cd) => !activeServiceDeviceIds.value.has(cd.deviceId))
    .map((cd) => devicesById.value.get(cd.deviceId))
    .filter((device): device is Device => !!device),
)

/**
 * Why "Add Service" is disabled, or null when it isn't -- shared by the
 * button's own disabled-reason (a title tooltip plus a visually-hidden
 * span, see BaseButton.vue's own doc comment) and a small always-visible
 * hint rendered next to the button in the template, since a hover-only
 * tooltip on a disabled (and therefore unfocusable) button is easy to
 * miss entirely.
 */
const addServiceDisabledReason = computed<string | null>(() => {
  if (locations.value.length === 0) return 'Add a location first'
  if (eligibleServiceDevices.value.length === 0) {
    return customerDevices.value.length === 0
      ? 'Attach a device first'
      : 'All attached devices already have a service'
  }
  return null
})

// --- Add/Remove Service ---

const showServiceForm = ref(false)
const serviceFormLocationId = ref('')

function openServiceForm() {
  serviceFormLocationId.value = locations.value[0]?.id ?? ''
  showServiceForm.value = true
}

/**
 * By the time ServiceFormDialog.vue emits 'created', the Service, its
 * ServiceEquipment link, and the real provision-service workflow against
 * the ONU have all already succeeded -- that whole action is
 * all-or-nothing (see that component's own doc comment): any failure
 * along the way rolls everything back and reports the error inline in
 * the still-open dialog instead of reaching this handler at all, so
 * there is no partial-success case to surface here anymore.
 */
async function handleServiceCreated(service: Service) {
  showServiceForm.value = false
  services.value = [...services.value, service]
  const labels = await resolveServiceLabels([service])
  serviceLabelsById.value = new Map(serviceLabelsById.value).set(service.id, labels.get(service.id) ?? service.id)
  await refreshActiveServiceEquipment() // the device this Service just claimed drops out of eligibleServiceDevices
}

const serviceDeleteTarget = ref<Service | null>(null)
const serviceDeletePending = ref(false)
const serviceDeleteError = ref<string | null>(null)

// Clears any error left over from a previous attempt (e.g. "still
// Active" before the operator went and suspended it) -- confirmDeleteService
// itself also clears it, but only once a new attempt actually runs, which
// left a stale error showing the instant this dialog reopens otherwise.
function openServiceDeleteDialog(service: Service) {
  serviceDeleteError.value = null
  serviceDeleteTarget.value = service
}

async function confirmDeleteService() {
  const target = serviceDeleteTarget.value
  if (!target) return
  serviceDeletePending.value = true
  serviceDeleteError.value = null
  try {
    // ServiceService.Delete blocks while Active (its config is still
    // applied on the OLT -- see Status.HasAppliedProfile's doc comment).
    // Remove Service used to just surface that as a dead-end error and
    // send the operator to the Service Workspace to run Suspend by hand
    // before coming back here -- the same "do X, remember to do Y" chain
    // this codebase prefers to collapse into one action rather than
    // patch the hand-off for. Suspended and Disconnected are "the
    // identical action at the Kontron config level" (status.go), so
    // suspending is enough to satisfy Delete's precondition; it's also
    // the one of the two already proven end-to-end from the UI
    // (ServiceDetailView.vue's primaryAction), unlike disconnect-service.
    if (target.status === 'Active') {
      const instance = await runWorkflow(target.id, 'suspend-service')
      if (instance.status !== 'Succeeded') {
        serviceDeleteError.value = instance.errorMessage ?? 'This service could not be suspended, so it was not removed.'
        return
      }
    }
    // Delete's second precondition: no active ServiceEquipment left
    // (internal/service/service.ServiceService.Delete only ever checks
    // ListActiveByServiceID, so an already-removed/historical row here
    // is left untouched). ServiceEquipmentService.Delete already tears
    // down its own AccessAttachment first (see that method's doc
    // comment), so removing each item is the same complete, self-
    // contained teardown ServiceDetailView.vue's own "Remove Equipment"
    // button performs -- just run for every attached item instead of
    // one at a time by hand.
    const activeEquipment = (await listServiceEquipmentByServiceId(target.id)).filter((item) => item.removedAt === null)
    for (const item of activeEquipment) {
      await deleteServiceEquipment(item.id)
    }
    await deleteService(target.id)
    services.value = services.value.filter((service) => service.id !== target.id)
    serviceDeleteTarget.value = null
    // Removing that equipment just freed its Device the same way
    // handleServiceCreated's own comment describes in reverse -- refresh
    // both the eligibility set and the Devices section's cached Status
    // (ServiceEquipmentService.Delete marks the Device Unused server-side)
    // rather than leave them showing stale Active state until a reload.
    if (activeEquipment.length > 0) {
      await Promise.all([refreshActiveServiceEquipment(), refreshAttachedDevices()])
    }
  } catch (err) {
    // A 404 here means the Service is already gone -- most often this
    // view's own `services` list going stale after another session (or
    // browser tab) deleted it first, not a real failure of this delete.
    // Reflect that instead of showing a confusing "could not be deleted"
    // for a Service that was, in fact, deleted.
    if (err instanceof ApiError && err.kind === 'not_found') {
      services.value = services.value.filter((service) => service.id !== target.id)
      serviceDeleteTarget.value = null
      return
    }
    // Conflict messages come straight from ServiceService.Delete now
    // (see that method's own doc comment): "still has equipment
    // attached" is a real, specific reason worth showing verbatim,
    // rather than collapsing it into one hardcoded guess. This also
    // catches a thrown runWorkflow error above (an ApiError from the
    // POST itself, or the plain Error it throws when the suspend
    // instance never reaches a terminal status in time) -- both carry a
    // real .message worth showing rather than the generic fallback,
    // same as ServiceDetailView.vue's own runAction catch.
    serviceDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? err.message
        : err instanceof Error
          ? err.message
          : 'The service could not be deleted.'
  } finally {
    serviceDeletePending.value = false
  }
}

const locationOptions = computed(() => locations.value.map((location) => ({ value: location.id, label: location.name })))

// Unlike locationOptions above (always a real choice, for Add Service),
// a CustomerDevice's Location is optional tracking (see
// CustomerDevice.locationId's own doc comment) -- "Not set" is a
// legitimate, common answer, not something to force a pick away from.
const deviceLocationOptions = computed(() => [{ value: '', label: 'Not set' }, ...locationOptions.value])

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
 * Which ONU Diagnostics blocks are collapsed, by serviceEquipmentId --
 * each block is independent (no accordion, more than one may be open at
 * once). A customer with several attached devices otherwise means
 * scrolling past one block's full result set (five commands' worth of
 * output) just to reach the next block's own "Check ONU Status" button;
 * this lets an operator collapse a block back down once they've seen
 * what they needed from it. Collapsing is purely a display toggle --
 * onuDiagnostics above still holds the results underneath, so expanding
 * again shows them instantly with no re-run.
 */
const collapsedONUDiagnostics = ref<Set<string>>(new Set())

function toggleONUDiagnosticsCollapsed(serviceEquipmentId: string) {
  const next = new Set(collapsedONUDiagnostics.value)
  if (next.has(serviceEquipmentId)) {
    next.delete(serviceEquipmentId)
  } else {
    next.add(serviceEquipmentId)
  }
  collapsedONUDiagnostics.value = next
}

/** The block header's large title -- resolved through deviceIdByServiceEquipmentId/devicesById rather than a bare interface string, so which physical ONU is which is legible at a glance. */
function onuDiagnosticsDeviceName(serviceEquipmentId: string): string {
  const deviceId = deviceIdByServiceEquipmentId.value.get(serviceEquipmentId)
  const device = deviceId ? devicesById.value.get(deviceId) : undefined
  return device?.name ?? 'Unknown Device'
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
  // A block collapsed from a previous run should not hide the fresh
  // results this run is about to produce.
  if (collapsedONUDiagnostics.value.has(equipmentLocation.serviceEquipmentId)) {
    const next = new Set(collapsedONUDiagnostics.value)
    next.delete(equipmentLocation.serviceEquipmentId)
    collapsedONUDiagnostics.value = next
  }

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
            <BaseButton variant="destructive" size="sm" @click="showRemoveCustomerDialog = true">
              Remove Customer
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

    <RemoveCustomerDialog
      :open="showRemoveCustomerDialog"
      :customer-id="customer.id"
      :customer-name="customer.name"
      @close="showRemoveCustomerDialog = false"
      @removed="handleCustomerRemoved"
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

    <SectionCard title="Devices" icon="devices" :badge="customerDevices.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showAttachDeviceDialog = true">Attach Device</BaseButton>
      </div>

      <AttachCustomerDeviceDialog
        :open="showAttachDeviceDialog"
        :customer-id="customer.id"
        :locations="locations"
        @close="showAttachDeviceDialog = false"
        @attached="handleDeviceAttached"
      />

      <ConfirmationDialog
        :open="deviceDetachTarget !== null"
        title="Detach Device"
        :description="`Detach ${deviceDetachTarget ? devicesById.get(deviceDetachTarget.deviceId)?.name ?? deviceDetachTarget.deviceId : ''} from this customer? The device record and its history stay -- Palladium never deletes it. This cannot be undone.`"
        confirm-label="Detach Device"
        destructive
        :pending="deviceDetachPending"
        :error="deviceDetachError"
        @confirm="confirmDetachDevice"
        @cancel="deviceDetachTarget = null"
      />

      <p v-if="deviceLocationError" class="device-location-error" role="alert">{{ deviceLocationError }}</p>

      <SimpleTable
        :columns="deviceColumns"
        :rows="customerDevices"
        :row-key="(record) => record.id"
        clickable
        empty-icon="devices"
        empty-title="No devices attached"
        @row-click="openDevice"
      >
        <template #cell-device="{ row }">
          <div class="device-cell">
            <span class="cell-strong">{{ devicesById.get(row.deviceId)?.name ?? row.deviceId }}</span>
            <span class="device-cell__meta">
              {{ devicesById.get(row.deviceId)?.manufacturer }} {{ devicesById.get(row.deviceId)?.model }} — Serial
              {{ devicesById.get(row.deviceId)?.serialNumber }}
            </span>
          </div>
        </template>
        <template #cell-location="{ row }">
          <BaseSelect
            label="Location"
            hide-label
            :model-value="row.locationId ?? ''"
            :options="deviceLocationOptions"
            :disabled="deviceLocationPending === row.id"
            @click.stop
            @update:model-value="(value) => handleDeviceLocationChanged(row, value)"
          />
        </template>
        <template #cell-status="{ row }">{{ devicesById.get(row.deviceId)?.status ?? '—' }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="deviceDetachTarget = row">Detach</BaseButton>
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
          :disabled="addServiceDisabledReason !== null"
          :disabled-reason="addServiceDisabledReason ?? undefined"
          @click="openServiceForm"
        >
          Add Service
        </BaseButton>
        <span v-if="addServiceDisabledReason" class="section-toolbar__hint">{{ addServiceDisabledReason }}</span>
      </div>

      <ServiceFormDialog
        :open="showServiceForm"
        :location-id="serviceFormLocationId"
        :devices="eligibleServiceDevices"
        :attached-device-count="customerDevices.length"
        @close="showServiceForm = false"
        @created="handleServiceCreated"
      />

      <ConfirmationDialog
        :open="serviceDeleteTarget !== null"
        title="Remove Service"
        :description="`Remove ${serviceDeleteTarget ? serviceLabelsById.get(serviceDeleteTarget.id) ?? serviceDeleteTarget.id : ''}? This will suspend it on the OLT first if it's Active, detach any assigned equipment, then delete the record. This cannot be undone.`"
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
        <template #cell-device="{ row }">{{ devicesById.get(serviceDeviceIdById.get(row.id) ?? '')?.name ?? '—' }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="openServiceDeleteDialog(row)">Remove</BaseButton>
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
          <button
            type="button"
            class="onu-diagnostics-block__toggle"
            :disabled="!onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.results"
            :aria-expanded="!collapsedONUDiagnostics.has(equipmentLocation.serviceEquipmentId)"
            @click="toggleONUDiagnosticsCollapsed(equipmentLocation.serviceEquipmentId)"
          >
            <BaseIcon
              v-if="onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.results"
              name="chevron-down"
              size="sm"
              class="onu-diagnostics-block__chevron"
              :class="{
                'onu-diagnostics-block__chevron--collapsed': collapsedONUDiagnostics.has(
                  equipmentLocation.serviceEquipmentId,
                ),
              }"
            />
            <span class="onu-diagnostics-block__title">
              <span class="onu-diagnostics-block__device">
                {{ onuDiagnosticsDeviceName(equipmentLocation.serviceEquipmentId) }}
              </span>
              <span class="onu-diagnostics-block__meta">
                {{ equipmentLocation.interface }} on {{ oltsById.get(equipmentLocation.oltId)?.name ?? equipmentLocation.oltId }}
              </span>
            </span>
          </button>
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

        <div
          v-if="
            onuDiagnostics.get(equipmentLocation.serviceEquipmentId)?.results &&
            !collapsedONUDiagnostics.has(equipmentLocation.serviceEquipmentId)
          "
          class="onu-diagnostics-results"
        >
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

    <SectionCard title="Notes" icon="notes" :badge="notes.length">
      <NotesSection :notes="notes" :submitting="notesSubmitting" :error="notesError" @submit="handleAddNote" />
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

.device-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.device-cell__meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.section-toolbar {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.section-toolbar__hint {
  padding-bottom: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
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

.onu-diagnostics-block__toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--color-text-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.onu-diagnostics-block__toggle:disabled {
  cursor: default;
}

.onu-diagnostics-block__chevron {
  flex-shrink: 0;
  color: var(--color-text-muted);
  transition: transform var(--motion-normal) var(--motion-ease);
}

.onu-diagnostics-block__chevron--collapsed {
  transform: rotate(-90deg);
}

.onu-diagnostics-block__title {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: var(--space-2);
  min-width: 0;
}

.onu-diagnostics-block__device {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.onu-diagnostics-block__meta {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

@media (prefers-reduced-motion: reduce) {
  .onu-diagnostics-block__chevron {
    transition: none;
  }
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

.device-location-error {
  margin: 0 0 var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
