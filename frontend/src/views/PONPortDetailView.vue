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
import AccessInterfaceFormDialog from '@/components/dialogs/AccessInterfaceFormDialog.vue'
import { getPONPortById, deletePONPort } from '@/services/ponPorts/ponPortRepository'
import { getOLTById } from '@/services/olts/oltRepository'
import { listAccessInterfacesByPONPortId, deleteAccessInterface } from '@/services/accessInterfaces/accessInterfaceRepository'
import { listEvents } from '@/services/events/eventRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { PONPort } from '@/types/ponPort'
import type { OLT } from '@/types/olt'
import type { AccessInterface } from '@/types/accessInterface'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The PON Port Detail Workspace -- deliberately the thinnest view in
 * this hierarchy: PONPort has no status field and almost no fields at
 * all (see types/ponPort.ts), but it still needs a page of its own
 * since Access Interfaces have to live somewhere. Same shape as
 * OLTDetailView.vue otherwise (single-relation OLT section, nested
 * Access Interfaces section (add/remove/open), delete-with-conflict-
 * handling).
 */
const route = useRoute()
const router = useRouter()

const ponPort = ref<PONPort | null>(null)
const olt = ref<OLT | null>(null)
const accessInterfaces = ref<AccessInterface[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  ponPort.value = null
  olt.value = null
  accessInterfaces.value = []
  timeline.value = []

  const result = await getPONPortById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  ponPort.value = result

  const [relatedOLT, ponPortAccessInterfaces, events] = await Promise.all([
    getOLTById(result.oltId),
    listAccessInterfacesByPONPortId(id),
    listEvents('pon_port', id),
  ])
  olt.value = relatedOLT
  accessInterfaces.value = ponPortAccessInterfaces
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const p = ponPort.value
  if (!p) return []
  return [
    { icon: 'clock', label: 'Created', value: formatDate(p.createdAt) },
    { icon: 'clock', label: 'Last Updated', value: formatDate(p.updatedAt) },
  ]
})

const accessInterfaceColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Access Interface' },
  { key: 'technology', label: 'Technology' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: '' },
]

function openAccessInterface(accessInterface: AccessInterface) {
  router.push(`/network/access-interfaces/${accessInterface.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Add/Remove Access Interface ---

const showAccessInterfaceForm = ref(false)

function handleAccessInterfaceCreated(accessInterface: AccessInterface) {
  showAccessInterfaceForm.value = false
  accessInterfaces.value = [...accessInterfaces.value, accessInterface]
}

const accessInterfaceDeleteTarget = ref<AccessInterface | null>(null)
const accessInterfaceDeletePending = ref(false)
const accessInterfaceDeleteError = ref<string | null>(null)

async function confirmDeleteAccessInterface() {
  const target = accessInterfaceDeleteTarget.value
  if (!target) return
  accessInterfaceDeletePending.value = true
  accessInterfaceDeleteError.value = null
  try {
    await deleteAccessInterface(target.id)
    accessInterfaces.value = accessInterfaces.value.filter((accessInterface) => accessInterface.id !== target.id)
    accessInterfaceDeleteTarget.value = null
  } catch (err) {
    accessInterfaceDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This access interface still has attachments — remove those first.'
        : 'The access interface could not be deleted.'
  } finally {
    accessInterfaceDeletePending.value = false
  }
}

// --- Delete PON Port ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeletePONPort() {
  if (!ponPort.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deletePONPort(ponPort.value.id)
    router.push(olt.value ? `/network/olts/${olt.value.id}` : '/network')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This PON port still has access interfaces attached — remove those first.'
        : 'The PON port could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="pon-port-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="pon-port-detail-view__status">
    <BaseErrorState title="PON port not found" description="This PON port may have been removed, or the link may be out of date.">
      <BaseButton variant="secondary" @click="router.push('/network')">Back to Network</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="ponPort">
    <WorkspaceHeader :title="`Port ${ponPort.portNumber}`" :metadata="[`PON Port ${ponPort.id}`]">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete PON Port</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete PON Port"
      :description="`Permanently delete Port ${ponPort.portNumber}? This cannot be undone.`"
      confirm-label="Delete PON Port"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeletePONPort"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="network">
      <FactGrid :facts="summaryFacts" />
      <p v-if="ponPort.description" class="pon-port-description">{{ ponPort.description }}</p>
    </SectionCard>

    <SectionCard title="OLT" icon="network">
      <RelationshipCard v-if="olt" eyebrow="OLT" :title="olt.name" :meta="olt.vendor" :to="`/network/olts/${olt.id}`" action-label="View OLT" />
      <p v-else class="no-relationship">No OLT on file for this PON port.</p>
    </SectionCard>

    <SectionCard title="Access Interfaces" icon="network" :badge="accessInterfaces.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showAccessInterfaceForm = true">Add Access Interface</BaseButton>
      </div>

      <AccessInterfaceFormDialog
        :open="showAccessInterfaceForm"
        :pon-port-id="ponPort.id"
        @close="showAccessInterfaceForm = false"
        @created="handleAccessInterfaceCreated"
      />

      <ConfirmationDialog
        :open="accessInterfaceDeleteTarget !== null"
        title="Remove Access Interface"
        :description="`Remove ${accessInterfaceDeleteTarget?.name}? This cannot be undone.`"
        confirm-label="Remove Access Interface"
        destructive
        :pending="accessInterfaceDeletePending"
        :error="accessInterfaceDeleteError"
        @confirm="confirmDeleteAccessInterface"
        @cancel="accessInterfaceDeleteTarget = null"
      />

      <SimpleTable
        :columns="accessInterfaceColumns"
        :rows="accessInterfaces"
        :row-key="(accessInterface) => accessInterface.id"
        clickable
        empty-icon="network"
        empty-title="No access interfaces on file"
        @row-click="openAccessInterface"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-technology="{ row }">{{ row.technology }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="accessInterfaceDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.pon-port-detail-view__status {
  padding: var(--space-6);
}

.pon-port-description {
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
