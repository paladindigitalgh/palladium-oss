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
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import SiteFormDialog from '@/components/dialogs/SiteFormDialog.vue'
import BuildingFormDialog from '@/components/dialogs/BuildingFormDialog.vue'
import { getSiteById, deleteSite } from '@/services/sites/siteRepository'
import { listBuildingsBySiteId, deleteBuilding } from '@/services/buildings/buildingRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Site } from '@/types/site'
import type { Building } from '@/types/building'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Site Detail Workspace, root of the Inventory hierarchy (Site ->
 * Building -> Room -> Rack -> Device). Mirrors AccessNetworkDetailView.vue's
 * shape -- Summary, a nested Buildings section (add/remove/open, same
 * treatment AccessNetworkDetailView gives OLTs), Timeline,
 * delete-with-conflict-handling.
 */
const route = useRoute()
const router = useRouter()

const site = ref<Site | null>(null)
const buildings = ref<Building[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  site.value = null
  buildings.value = []
  timeline.value = []

  const result = await getSiteById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  site.value = result

  const [siteBuildings, events] = await Promise.all([listBuildingsBySiteId(id), listEvents('site', id)])
  buildings.value = siteBuildings
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const s = site.value
  if (!s) return []
  return [
    { icon: 'clock', label: 'Created', value: formatDate(s.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(s.updatedAt) },
  ]
})

const buildingColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Building' },
  { key: 'actions', label: '' },
]

function openBuilding(building: Building) {
  router.push(`/inventory/buildings/${building.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Add/Remove Building ---

const showBuildingForm = ref(false)

function handleBuildingCreated(building: Building) {
  showBuildingForm.value = false
  buildings.value = [...buildings.value, building]
}

const buildingDeleteTarget = ref<Building | null>(null)
const buildingDeletePending = ref(false)
const buildingDeleteError = ref<string | null>(null)

async function confirmDeleteBuilding() {
  const target = buildingDeleteTarget.value
  if (!target) return
  buildingDeletePending.value = true
  buildingDeleteError.value = null
  try {
    await deleteBuilding(target.id)
    buildings.value = buildings.value.filter((building) => building.id !== target.id)
    buildingDeleteTarget.value = null
  } catch (err) {
    buildingDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This building still has rooms attached — remove those first.'
        : 'The building could not be deleted.'
  } finally {
    buildingDeletePending.value = false
  }
}

// --- Edit Site ---

const showEditDialog = ref(false)

function handleSiteUpdated(updated: Site) {
  site.value = updated
  showEditDialog.value = false
}

// --- Delete Site ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteSite() {
  if (!site.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteSite(site.value.id)
    router.push('/inventory')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This site still has buildings attached — remove those first.'
        : 'The site could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="site-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="site-detail-view__status">
    <BaseErrorState title="Site not found" description="This site may have been removed, or the link may be out of date.">
      <BaseButton variant="secondary" @click="router.push('/inventory')">Back to Inventory</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="site">
    <WorkspaceHeader :title="site.name" :metadata="[`Site ${site.id}`]">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Site</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Site</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SiteFormDialog :open="showEditDialog" :site="site" @close="showEditDialog = false" @updated="handleSiteUpdated" />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Site"
      :description="`Permanently delete ${site.name}? This cannot be undone.`"
      confirm-label="Delete Site"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteSite"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="inventory">
      <FactGrid :facts="summaryFacts" />
      <p v-if="site.description" class="site-description">{{ site.description }}</p>
    </SectionCard>

    <SectionCard title="Buildings" icon="inventory" :badge="buildings.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showBuildingForm = true">Add Building</BaseButton>
      </div>

      <BuildingFormDialog
        :open="showBuildingForm"
        :site-id="site.id"
        @close="showBuildingForm = false"
        @created="handleBuildingCreated"
      />

      <ConfirmationDialog
        :open="buildingDeleteTarget !== null"
        title="Remove Building"
        :description="`Remove ${buildingDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove Building"
        destructive
        :pending="buildingDeletePending"
        :error="buildingDeleteError"
        @confirm="confirmDeleteBuilding"
        @cancel="buildingDeleteTarget = null"
      />

      <SimpleTable
        :columns="buildingColumns"
        :rows="buildings"
        :row-key="(building) => building.id"
        clickable
        empty-icon="inventory"
        empty-title="No buildings on file"
        @row-click="openBuilding"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="buildingDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.site-detail-view__status {
  padding: var(--space-6);
}

.site-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.section-toolbar {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
</style>
