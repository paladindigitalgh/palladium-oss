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
import BuildingFormDialog from '@/components/dialogs/BuildingFormDialog.vue'
import { getBuildingById, deleteBuilding } from '@/services/buildings/buildingRepository'
import { getSiteById } from '@/services/sites/siteRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Building } from '@/types/building'
import type { Site } from '@/types/site'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Building Detail Workspace. Mirrors OLTDetailView.vue's shape for
 * the single-relation Site section (a RelationshipCard, resolved on
 * demand) and delete-with-conflict-handling. The Rooms section is added
 * in a follow-up commit alongside RoomFormDialog/roomRepository.ts, the
 * same staging OLTDetailView.vue's own PON Ports section used relative
 * to AccessNetworkDetailView.vue.
 */
const route = useRoute()
const router = useRouter()

const building = ref<Building | null>(null)
const site = ref<Site | null>(null)
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  building.value = null
  site.value = null
  timeline.value = []

  const result = await getBuildingById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  building.value = result

  const [relatedSite, events] = await Promise.all([getSiteById(result.siteId), listEvents('building', id)])
  site.value = relatedSite
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

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

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
</style>
