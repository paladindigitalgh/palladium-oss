<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import RelationshipCard from '@/components/data-display/RelationshipCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import RoomFormDialog from '@/components/dialogs/RoomFormDialog.vue'
import { getRoomById, deleteRoom } from '@/services/rooms/roomRepository'
import { getBuildingById } from '@/services/buildings/buildingRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Room } from '@/types/room'
import type { Building } from '@/types/building'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Room Detail Workspace. Mirrors BuildingDetailView.vue's shape for
 * the single-relation Building section and delete-with-conflict-handling.
 * The Racks section is added in a follow-up commit alongside
 * RackFormDialog/rackRepository.ts, the same staging BuildingDetailView.vue's
 * own Rooms section used relative to SiteDetailView.vue.
 */
const route = useRoute()
const router = useRouter()

const room = ref<Room | null>(null)
const building = ref<Building | null>(null)
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  room.value = null
  building.value = null
  timeline.value = []

  const result = await getRoomById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  room.value = result

  const [relatedBuilding, events] = await Promise.all([getBuildingById(result.buildingId), listEvents('room', id)])
  building.value = relatedBuilding
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const r = room.value
  if (!r) return []
  return [
    { icon: 'clock', label: 'Created', value: formatDate(r.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(r.updatedAt) },
  ]
})

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Edit Room ---

const showEditDialog = ref(false)

function handleRoomUpdated(updated: Room) {
  room.value = updated
  showEditDialog.value = false
}

// --- Delete Room ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteRoom() {
  if (!room.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteRoom(room.value.id)
    router.push(building.value ? `/inventory/buildings/${building.value.id}` : '/inventory')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This room still has racks attached — remove those first.'
        : 'The room could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="room-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="room-detail-view__status">
    <BaseErrorState
      title="Room not found"
      description="This room may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/inventory')">Back to Inventory</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="room">
    <WorkspaceHeader :title="room.name" :metadata="[`Room ${room.id}`]">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Room</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Room</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <RoomFormDialog
      :open="showEditDialog"
      :building-id="room.buildingId"
      :room="room"
      @close="showEditDialog = false"
      @updated="handleRoomUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Room"
      :description="`Permanently delete ${room.name}? This cannot be undone.`"
      confirm-label="Delete Room"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteRoom"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="inventory">
      <FactGrid :facts="summaryFacts" />
      <p v-if="room.description" class="room-description">{{ room.description }}</p>
    </SectionCard>

    <SectionCard title="Building" icon="inventory">
      <RelationshipCard
        v-if="building"
        eyebrow="Building"
        :title="building.name"
        :to="`/inventory/buildings/${building.id}`"
        action-label="View Building"
      />
      <p v-else class="no-relationship">No building on file for this room.</p>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.room-detail-view__status {
  padding: var(--space-6);
}

.room-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
