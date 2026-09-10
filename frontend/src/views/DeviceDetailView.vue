<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import RelationshipCard from '@/components/data-display/RelationshipCard.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import NotesSection from '@/components/data-display/NotesSection.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import DeviceFormDialog from '@/components/dialogs/DeviceFormDialog.vue'
import { getDeviceById } from '@/services/devices/deviceRepository'
import { listServiceEquipmentByDeviceId } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { deauthorizeONU } from '@/services/provisioning/provisioningRepository'
import { getServiceById } from '@/services/services/serviceRepository'
import { getRackById } from '@/services/racks/rackRepository'
import { listEvents } from '@/services/events/eventRepository'
import { listNotes, createNote } from '@/services/notes/noteRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { Service } from '@/types/service'
import type { Rack } from '@/types/rack'
import type { TimelineEvent } from '@/types/timelineEvent'
import type { Note } from '@/types/note'

/**
 * The Device Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 10, "Device Workspace"), backed by the real Inventory API.
 *
 * Sections that depended on mock-only telemetry concepts (Network,
 * Status, Configuration) are removed rather than faked -- see
 * types/device.ts's doc comment for why (Palladium is not a monitoring
 * platform). Assignment is real: resolved on demand via
 * ServiceEquipment (docs/03-DOMAIN-MODEL.md -- a Device's relationship to
 * a Customer always passes through Service, never a direct link), and a
 * Device can have zero, one, or more equipment assignments over its
 * lifetime, so every one that comes back is shown, not just the first.
 */
const route = useRoute()
const router = useRouter()

const device = ref<Device | null>(null)
const assignedServices = ref<Service[]>([])
const rack = ref<Rack | null>(null)
const timeline = ref<TimelineEvent[]>([])
const notes = ref<Note[]>([])
const notesSubmitting = ref(false)
const notesError = ref<string | null>(null)
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  device.value = null
  assignedServices.value = []
  rack.value = null
  timeline.value = []
  notes.value = []

  const result = await getDeviceById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  device.value = result

  const [equipment, events, deviceNotes, deviceRack] = await Promise.all([
    listServiceEquipmentByDeviceId(id),
    listEvents('device', id),
    listNotes('device', id),
    result.rackId ? getRackById(result.rackId) : Promise.resolve(null),
  ])
  timeline.value = events
  notes.value = deviceNotes
  rack.value = deviceRack

  const services = await Promise.all(equipment.map((item) => getServiceById(item.serviceId)))
  assignedServices.value = services.filter((service): service is Service => service !== null)

  loading.value = false
}

async function refreshNotes() {
  if (!device.value) return
  notes.value = await listNotes('device', device.value.id)
}

async function handleAddNote(body: string) {
  if (!device.value) return
  notesSubmitting.value = true
  notesError.value = null
  try {
    await createNote({ entityType: 'device', entityId: device.value.id, body })
    await refreshNotes()
  } catch {
    notesError.value = 'The note could not be added.'
  } finally {
    notesSubmitting.value = false
  }
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: Fact[] = [
    { icon: 'inventory', label: 'Manufacturer', value: d.manufacturer },
    { icon: 'devices', label: 'Model', value: d.model },
  ]
  if (d.assetTag) facts.push({ icon: 'tasks', label: 'Asset Tag', value: d.assetTag })
  facts.push(
    { icon: 'clock', label: 'Created', value: formatDate(d.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(d.updatedAt) },
  )
  return facts
})

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

const headerMetadata = computed<string[]>(() => {
  const d = device.value
  if (!d) return []
  const entries = [`Serial ${d.serialNumber}`]
  if (d.assetTag) entries.push(`Asset Tag ${d.assetTag}`)
  return entries
})

// --- Edit Device ---

const showEditDialog = ref(false)

function handleDeviceUpdated(updated: Device) {
  device.value = updated
  showEditDialog.value = false
}

// --- Remove Device ---
// "Remove Device" in the UI, internal/provisioning/kontron/service.
// DeauthorizationService's Deauthorize ONU underneath: it removes the
// ONU's authorization from its real OLT, unassigns it from its current
// Service, and marks the Device Retired -- it never erases the row.
// Palladium has no permanent-delete action for a Device at all (see
// inventory.DeviceRepository's own doc comment for why): history (past
// Service assignments, timeline) is always preserved, so Remove Device
// is the one and only way to take a Device out of service.

/**
 * Offered for any device not already Retired: a Device can be
 * OLT-authorized while Unused (freshly discovered/authorized, not yet
 * attached to a Customer) or Active (attached), and either one might
 * genuinely still be live on its OLT -- only Retired means it has
 * already been fully deauthorized. internal/provisioning/kontron/
 * service.DeauthorizationService resolves the rest (equipment or
 * OnuAuthorization record, OLT, interface) server-side, and errors
 * cleanly if that resolution fails -- this is just the cheap
 * client-side gate that avoids offering the action when it plainly
 * cannot apply.
 */
const canRemoveDevice = computed(() => !!device.value && device.value.status !== 'Retired')

const showRemoveDialog = ref(false)
const removePending = ref(false)
const removeError = ref<string | null>(null)

async function confirmRemoveDevice() {
  if (!device.value) return
  removePending.value = true
  removeError.value = null
  try {
    await deauthorizeONU(device.value.id)
    showRemoveDialog.value = false
    // Unlike Edit or a failed removal, a successful Remove Device leaves
    // nothing on this page worth staying for -- the Device is now
    // Retired, and the operator's next move is almost always back to the
    // list, not watching this one record reload in place.
    router.push('/devices')
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'invalid') {
      removeError.value = 'This device is not an ONU/ONT on a Kontron OLT.'
    } else if (err instanceof ApiError && err.kind === 'not_found') {
      removeError.value = 'This device has never been authorized through Palladium, so there is nothing to remove.'
    } else {
      removeError.value = err instanceof ApiError ? err.message : 'The device could not be removed.'
    }
  } finally {
    removePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="device-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="device-detail-view__status">
    <BaseErrorState
      title="Device not found"
      description="This device may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/devices')">Back to Devices</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="device">
    <WorkspaceHeader
      :title="device.name"
      :subtitle="`${device.manufacturer} ${device.model}`"
      :status="{ label: device.status, variant: device.status === 'Active' ? 'success' : 'neutral' }"
      :metadata="headerMetadata"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Device</BaseButton>
            <BaseButton v-if="canRemoveDevice" variant="destructive" size="sm" @click="showRemoveDialog = true">
              Remove Device
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <DeviceFormDialog :open="showEditDialog" :device="device" @close="showEditDialog = false" @updated="handleDeviceUpdated" />

    <ConfirmationDialog
      :open="showRemoveDialog"
      title="Remove Device"
      :description="`Remove ${device.name} (Serial ${device.serialNumber}) from its OLT and unassign it from its current service, then mark the Device Retired. The inventory record stays -- Palladium never deletes a Device's history. This cannot be undone.`"
      confirm-label="Remove Device"
      destructive
      :pending="removePending"
      :error="removeError"
      @confirm="confirmRemoveDevice"
      @cancel="showRemoveDialog = false"
    />

    <SectionCard title="Summary" icon="devices">
      <FactGrid :facts="summaryFacts" />
      <p v-if="device.description" class="device-description">{{ device.description }}</p>
      <RelationshipCard
        v-if="rack"
        class="device-rack-card"
        eyebrow="Rack"
        :title="rack.name"
        :to="`/administration/inventory/racks/${rack.id}`"
        action-label="View Rack"
      />
    </SectionCard>

    <SectionCard title="Assignment" icon="services" :badge="assignedServices.length">
      <BaseEmptyState
        v-if="assignedServices.length === 0"
        icon="devices"
        title="Not currently assigned to a service"
        description="This device is not fulfilling any Service right now."
      />
      <div v-else class="assignment-cards">
        <RelationshipCard
          v-for="service in assignedServices"
          :key="service.id"
          eyebrow="Assigned Service"
          :title="service.description || `Service ${service.id}`"
          :meta="service.status"
          :to="`/services/${service.id}`"
          action-label="View Service"
        />
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
.device-detail-view__status {
  padding: var(--space-6);
}

.device-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.device-rack-card {
  margin-top: var(--space-4);
}

.assignment-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--space-4);
}
</style>
