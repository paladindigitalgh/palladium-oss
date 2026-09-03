<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import SiteFormDialog from '@/components/dialogs/SiteFormDialog.vue'
import { getSiteById, deleteSite } from '@/services/sites/siteRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Site } from '@/types/site'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Site Detail Workspace, root of the Inventory hierarchy (Site ->
 * Building -> Room -> Rack -> Device). Mirrors AccessNetworkDetailView.vue's
 * shape -- Summary, Timeline, delete-with-conflict-handling. The
 * Buildings section is added in a follow-up commit alongside
 * BuildingFormDialog/buildingRepository.ts, the same staging
 * AccessNetworkDetailView.vue's OLTs section used.
 */
const route = useRoute()
const router = useRouter()

const site = ref<Site | null>(null)
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  site.value = null
  timeline.value = []

  const result = await getSiteById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  site.value = result

  timeline.value = await listEvents('site', id)

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

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

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
</style>
