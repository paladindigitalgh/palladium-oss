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
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import BuildingFormDialog from '@/components/dialogs/BuildingFormDialog.vue'
import RoomFormDialog from '@/components/dialogs/RoomFormDialog.vue'
import { getBuildingById, deleteBuilding } from '@/services/buildings/buildingRepository'
import { getSiteById } from '@/services/sites/siteRepository'
import { listRoomsByBuildingId, deleteRoom } from '@/services/rooms/roomRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Building } from '@/types/building'
import type { Site } from '@/types/site'
import type { Room } from '@/types/room'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Building Detail Workspace. Mirrors SiteDetailView.vue's shape for
 * the single-relation Site section (a RelationshipCard, resolved on
 * demand), a nested Rooms section (add/remove/open, same treatment
 * SiteDetailView gives Buildings), and delete-with-conflict-handling.
 */
const route = useRoute()
const router = useRouter()

const building = ref<Building | null>(null)
const site = ref<Site | null>(null)
const rooms = ref<Room[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  building.value = null
  site.value = null
  rooms.value = []
  timeline.value = []

  const result = await getBuildingById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  building.value = result

  const [relatedSite, buildingRooms, events] = await Promise.all([
    getSiteById(result.siteId),
    listRoomsByBuildingId(id),
    listEvents('building', id),
  ])
  site.value = relatedSite
  rooms.value = buildingRooms
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const b = building.value
  if (!b) return []
  return [
    { icon: 'clock', label: 'Created', value: formatDate(b.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(b.updatedAt) },
  ]
})

const roomColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Room' },
  { key: 'actions', label: '' },
]

function openRoom(room: Room) {
  router.push(`/inventory/rooms/${room.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Add/Remove Room ---

const showRoomForm = ref(false)

function handleRoomCreated(room: Room) {
  showRoomForm.value = false
  rooms.value = [...rooms.value, room]
}

const roomDeleteTarget = ref<Room | null>(null)
const roomDeletePending = ref(false)
const roomDeleteError = ref<string | null>(null)

async function confirmDeleteRoom() {
  const target = roomDeleteTarget.value
  if (!target) return
  roomDeletePending.value = true
  roomDeleteError.value = null
  try {
    await deleteRoom(target.id)
    rooms.value = rooms.value.filter((room) => room.id !== target.id)
    roomDeleteTarget.value = null
  } catch (err) {
    roomDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This room still has racks attached — remove those first.'
        : 'The room could not be deleted.'
  } finally {
    roomDeletePending.value = false
  }
}

// --- Edit Building ---

const showEditDialog = ref(false)

function handleBuildingUpdated(updated: Building) {
  building.value = updated
  showEditDialog.value = false
}

// --- Delete Building ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteBuilding() {
  if (!building.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteBuilding(building.value.id)
    router.push(site.value ? `/inventory/${site.value.id}` : '/inventory')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This building still has rooms attached — remove those first.'
        : 'The building could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="building-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="building-detail-view__status">
    <BaseErrorState
      title="Building not found"
      description="This building may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/inventory')">Back to Inventory</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="building">
    <WorkspaceHeader :title="building.name" :metadata="[`Building ${building.id}`]">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Building</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Building</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <BuildingFormDialog
      :open="showEditDialog"
      :site-id="building.siteId"
      :building="building"
      @close="showEditDialog = false"
      @updated="handleBuildingUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Building"
      :description="`Permanently delete ${building.name}? This cannot be undone.`"
      confirm-label="Delete Building"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteBuilding"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="inventory">
      <FactGrid :facts="summaryFacts" />
      <p v-if="building.description" class="building-description">{{ building.description }}</p>
    </SectionCard>

    <SectionCard title="Site" icon="inventory">
      <RelationshipCard
        v-if="site"
        eyebrow="Site"
        :title="site.name"
        :to="`/inventory/${site.id}`"
        action-label="View Site"
      />
      <p v-else class="no-relationship">No site on file for this building.</p>
    </SectionCard>

    <SectionCard title="Rooms" icon="inventory" :badge="rooms.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showRoomForm = true">Add Room</BaseButton>
      </div>

      <RoomFormDialog
        :open="showRoomForm"
        :building-id="building.id"
        @close="showRoomForm = false"
        @created="handleRoomCreated"
      />

      <ConfirmationDialog
        :open="roomDeleteTarget !== null"
        title="Remove Room"
        :description="`Remove ${roomDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove Room"
        destructive
        :pending="roomDeletePending"
        :error="roomDeleteError"
        @confirm="confirmDeleteRoom"
        @cancel="roomDeleteTarget = null"
      />

      <SimpleTable
        :columns="roomColumns"
        :rows="rooms"
        :row-key="(room) => room.id"
        clickable
        empty-icon="inventory"
        empty-title="No rooms on file"
        @row-click="openRoom"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="roomDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.building-detail-view__status {
  padding: var(--space-6);
}

.building-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.section-toolbar {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
</style>
