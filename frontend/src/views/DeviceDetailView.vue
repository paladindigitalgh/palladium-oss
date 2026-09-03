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
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import DeviceFormDialog from '@/components/dialogs/DeviceFormDialog.vue'
import { getDeviceById, deleteDevice } from '@/services/devices/deviceRepository'
import { listServiceEquipmentByDeviceId } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { getServiceById } from '@/services/services/serviceRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Device } from '@/types/device'
import type { Service } from '@/types/service'
import type { TimelineEvent } from '@/types/timelineEvent'

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
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  device.value = null
  assignedServices.value = []
  timeline.value = []

  const result = await getDeviceById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  device.value = result

  const [equipment, events] = await Promise.all([listServiceEquipmentByDeviceId(id), listEvents('device', id)])
  timeline.value = events

  const services = await Promise.all(equipment.map((item) => getServiceById(item.serviceId)))
  assignedServices.value = services.filter((service): service is Service => service !== null)

  loading.value = false
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

// --- Delete Device ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteDevice() {
  if (!device.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteDevice(device.value.id)
    router.push('/devices')
  } catch (err) {
    deleteError.value = err instanceof ApiError ? err.message : 'The device could not be deleted.'
  } finally {
    deletePending.value = false
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
      :status="{ label: device.status, variant: device.status === 'Installed' || device.status === 'InStock' ? 'success' : 'neutral' }"
      :metadata="headerMetadata"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Device</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Device</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <DeviceFormDialog :open="showEditDialog" :device="device" @close="showEditDialog = false" @updated="handleDeviceUpdated" />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Device"
      :description="`Permanently delete ${device.name}? This cannot be undone.`"
      confirm-label="Delete Device"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteDevice"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="devices">
      <FactGrid :facts="summaryFacts" />
      <p v-if="device.description" class="device-description">{{ device.description }}</p>
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

.assignment-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--space-4);
}
</style>
