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
import OLTFormDialog from '@/components/dialogs/OLTFormDialog.vue'
import { getAccessNetworkById, deleteAccessNetwork } from '@/services/accessNetworks/accessNetworkRepository'
import { listOLTsByAccessNetworkId, deleteOLT } from '@/services/olts/oltRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { AccessNetwork } from '@/types/accessNetwork'
import type { OLT } from '@/types/olt'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Access Network Detail Workspace, root of the access-network
 * hierarchy (AccessNetwork -> OLT -> PONPort -> AccessInterface ->
 * AccessAttachment). Mirrors CustomerDetailView.vue's shape -- Summary,
 * a nested OLTs section (add/remove/open, same treatment
 * CustomerDetailView gives Locations/Services), Timeline,
 * delete-with-conflict-handling.
 */
const route = useRoute()
const router = useRouter()

const accessNetwork = ref<AccessNetwork | null>(null)
const olts = ref<OLT[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  accessNetwork.value = null
  olts.value = []
  timeline.value = []

  const result = await getAccessNetworkById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  accessNetwork.value = result

  const [accessNetworkOLTs, events] = await Promise.all([listOLTsByAccessNetworkId(id), listEvents('access_network', id)])
  olts.value = accessNetworkOLTs
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const a = accessNetwork.value
  if (!a) return []
  return [
    { icon: 'health', label: 'Status', value: a.status },
    { icon: 'clock', label: 'Created', value: formatDate(a.createdAt) },
  ]
})

const oltColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'OLT' },
  { key: 'vendor', label: 'Vendor / Model' },
  { key: 'actions', label: '' },
]

function openOLT(olt: OLT) {
  router.push(`/network/olts/${olt.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Add/Remove OLT ---

const showOLTForm = ref(false)

function handleOLTCreated(olt: OLT) {
  showOLTForm.value = false
  olts.value = [...olts.value, olt]
}

const oltDeleteTarget = ref<OLT | null>(null)
const oltDeletePending = ref(false)
const oltDeleteError = ref<string | null>(null)

async function confirmDeleteOLT() {
  const target = oltDeleteTarget.value
  if (!target) return
  oltDeletePending.value = true
  oltDeleteError.value = null
  try {
    await deleteOLT(target.id)
    olts.value = olts.value.filter((olt) => olt.id !== target.id)
    oltDeleteTarget.value = null
  } catch (err) {
    oltDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This OLT still has PON ports attached — remove those first.'
        : 'The OLT could not be deleted.'
  } finally {
    oltDeletePending.value = false
  }
}

// --- Delete Access Network ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteAccessNetwork() {
  if (!accessNetwork.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteAccessNetwork(accessNetwork.value.id)
    router.push('/network')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This access network still has OLTs attached — remove those first.'
        : 'The access network could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="access-network-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="access-network-detail-view__status">
    <BaseErrorState
      title="Access network not found"
      description="This access network may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/network')">Back to Network</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="accessNetwork">
    <WorkspaceHeader
      :title="accessNetwork.name"
      :status="{ label: accessNetwork.status, variant: accessNetwork.status === 'Active' ? 'success' : 'neutral' }"
      :metadata="[`Access Network ${accessNetwork.id}`]"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">
              Delete Access Network
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Access Network"
      :description="`Permanently delete ${accessNetwork.name}? This cannot be undone.`"
      confirm-label="Delete Access Network"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteAccessNetwork"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="network">
      <FactGrid :facts="summaryFacts" />
      <p v-if="accessNetwork.description" class="access-network-description">{{ accessNetwork.description }}</p>
    </SectionCard>

    <SectionCard title="OLTs" icon="network" :badge="olts.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showOLTForm = true">Add OLT</BaseButton>
      </div>

      <OLTFormDialog
        :open="showOLTForm"
        :access-network-id="accessNetwork.id"
        @close="showOLTForm = false"
        @created="handleOLTCreated"
      />

      <ConfirmationDialog
        :open="oltDeleteTarget !== null"
        title="Remove OLT"
        :description="`Remove ${oltDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove OLT"
        destructive
        :pending="oltDeletePending"
        :error="oltDeleteError"
        @confirm="confirmDeleteOLT"
        @cancel="oltDeleteTarget = null"
      />

      <SimpleTable
        :columns="oltColumns"
        :rows="olts"
        :row-key="(olt) => olt.id"
        clickable
        empty-icon="network"
        empty-title="No OLTs on file"
        @row-click="openOLT"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-vendor="{ row }">{{ row.vendor }} {{ row.model }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="oltDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.access-network-detail-view__status {
  padding: var(--space-6);
}

.access-network-description {
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
