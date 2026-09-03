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
import RelationshipCard from '@/components/data-display/RelationshipCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import RackFormDialog from '@/components/dialogs/RackFormDialog.vue'
import { getRackById, deleteRack } from '@/services/racks/rackRepository'
import { getRoomById } from '@/services/rooms/roomRepository'
import { listDevicesByRackId } from '@/services/devices/deviceRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Rack } from '@/types/rack'
import type { Room } from '@/types/room'
import type { Device, DeviceStatus } from '@/types/device'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Rack Detail Workspace, the bottom of the Inventory hierarchy.
 * Mirrors RoomDetailView.vue's shape for the single-relation parent
 * section, except Room is nullable here (a Rack can be unassigned) and
 * the RelationshipCard is skipped -- rather than shown as broken -- when
 * there is none. The Devices section is read-only (rows link to
 * /devices/:id, mirroring DeviceCollectionView.vue's row shape): Racks
 * do not add or remove Devices themselves, a Device chooses its Rack
 * from DeviceFormDialog.vue's Rack picker instead.
 */
const route = useRoute()
const router = useRouter()

const rack = ref<Rack | null>(null)
const room = ref<Room | null>(null)
const devices = ref<Device[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  rack.value = null
  room.value = null
  devices.value = []
  timeline.value = []

  const result = await getRackById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  rack.value = result

  const [relatedRoom, rackDevices, events] = await Promise.all([
    result.roomId ? getRoomById(result.roomId) : Promise.resolve(null),
    listDevicesByRackId(id),
    listEvents('rack', id),
  ])
  room.value = relatedRoom
  devices.value = rackDevices
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const rk = rack.value
  if (!rk) return []
  return [
    { icon: 'clock', label: 'Created', value: formatDate(rk.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(rk.updatedAt) },
  ]
})

const deviceColumns: SimpleTableColumn[] = [
  { key: 'device', label: 'Device' },
  { key: 'status', label: 'Status' },
]

const STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info' | 'neutral'> = {
  Ordered: 'info',
  Received: 'info',
  InStock: 'success',
  Installed: 'success',
  Maintenance: 'warning',
  Retired: 'neutral',
  Disposed: 'neutral',
}

function openDevice(device: Device) {
  router.push(`/devices/${device.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Edit Rack ---

const showEditDialog = ref(false)

function handleRackUpdated(updated: Rack) {
  rack.value = updated
  showEditDialog.value = false
}

// --- Delete Rack ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteRack() {
  if (!rack.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteRack(rack.value.id)
    router.push(room.value ? `/inventory/rooms/${room.value.id}` : '/inventory')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This rack still has devices attached — remove those first.'
        : 'The rack could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="rack-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="rack-detail-view__status">
    <BaseErrorState
      title="Rack not found"
      description="This rack may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/inventory')">Back to Inventory</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="rack">
    <WorkspaceHeader :title="rack.name" :metadata="[`Rack ${rack.id}`]">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Rack</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Rack</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <RackFormDialog
      :open="showEditDialog"
      :room-id="rack.roomId ?? ''"
      :rack="rack"
      @close="showEditDialog = false"
      @updated="handleRackUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Rack"
      :description="`Permanently delete ${rack.name}? This cannot be undone.`"
      confirm-label="Delete Rack"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteRack"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="inventory">
      <FactGrid :facts="summaryFacts" />
      <p v-if="rack.description" class="rack-description">{{ rack.description }}</p>
    </SectionCard>

    <SectionCard title="Room" icon="inventory">
      <RelationshipCard
        v-if="room"
        eyebrow="Room"
        :title="room.name"
        :to="`/inventory/rooms/${room.id}`"
        action-label="View Room"
      />
      <p v-else class="no-relationship">This rack is not assigned to a room.</p>
    </SectionCard>

    <SectionCard title="Devices" icon="devices" :badge="devices.length">
      <SimpleTable
        :columns="deviceColumns"
        :rows="devices"
        :row-key="(device) => device.id"
        clickable
        empty-icon="devices"
        empty-title="No devices racked here"
        @row-click="openDevice"
      >
        <template #cell-device="{ row }">{{ row.name }}</template>
        <template #cell-status="{ row }">
          <BaseBadge :variant="STATUS_VARIANTS[row.status as DeviceStatus]">{{ row.status }}</BaseBadge>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.rack-detail-view__status {
  padding: var(--space-6);
}

.rack-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
