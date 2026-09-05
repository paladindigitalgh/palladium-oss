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
import OLTFormDialog from '@/components/dialogs/OLTFormDialog.vue'
import PONPortFormDialog from '@/components/dialogs/PONPortFormDialog.vue'
import { getOLTById, deleteOLT } from '@/services/olts/oltRepository'
import { getAccessNetworkById } from '@/services/accessNetworks/accessNetworkRepository'
import { listPONPortsByOLTId, deletePONPort } from '@/services/ponPorts/ponPortRepository'
import { listEvents } from '@/services/events/eventRepository'
import { runONUSummary, runONUStatusSummary } from '@/services/diagnostics/diagnosticsRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { OLT } from '@/types/olt'
import type { AccessNetwork } from '@/types/accessNetwork'
import type { PONPort } from '@/types/ponPort'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The OLT Detail Workspace. Mirrors ServiceDetailView.vue's shape for the
 * single-relation Access Network section (a RelationshipCard, resolved
 * on demand) and CustomerDetailView.vue's shape for the nested PON Ports
 * section (add/remove/open), delete-with-conflict-handling, and the ONU
 * Status section's raw-output rendering (no parsing anywhere in this
 * stack -- see internal/diagnostics/kontron's own doc comment).
 *
 * Unlike Customer Detail's per-equipment "Check ONU Status" (four/five
 * commands run against one known interface), this runs the OLT's own
 * whole-device summary commands ("show onu interface all" and
 * "...all status") -- an operator-triggered, on-demand snapshot, not
 * continuous polling or alarms, so it does not cross into the
 * monitoring-platform territory docs/09-WORKSPACE-SPECIFICATIONS.md
 * section 11 deliberately keeps off this workspace.
 */
const route = useRoute()
const router = useRouter()

const olt = ref<OLT | null>(null)
const accessNetwork = ref<AccessNetwork | null>(null)
const ponPorts = ref<PONPort[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  olt.value = null
  accessNetwork.value = null
  ponPorts.value = []
  timeline.value = []
  onuStatusResults.value = null

  const result = await getOLTById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  olt.value = result

  const [relatedAccessNetwork, oltPONPorts, events] = await Promise.all([
    getAccessNetworkById(result.accessNetworkId),
    listPONPortsByOLTId(id),
    listEvents('olt', id),
  ])
  accessNetwork.value = relatedAccessNetwork
  ponPorts.value = oltPONPorts
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const o = olt.value
  if (!o) return []
  const facts: Fact[] = [
    { icon: 'network', label: 'Vendor', value: o.vendor },
    { icon: 'network', label: 'Model', value: o.model || '—' },
  ]
  if (o.managementIpAddress) facts.push({ icon: 'network', label: 'Management IP', value: o.managementIpAddress })
  facts.push({ icon: 'clock', label: 'Created', value: formatDate(o.createdAt) })
  return facts
})

const headerMetadata = computed<string[]>(() => {
  const o = olt.value
  if (!o) return []
  return o.managementIpAddress ? [`Management IP ${o.managementIpAddress}`] : []
})

const ponPortColumns: SimpleTableColumn[] = [
  { key: 'port', label: 'Port' },
  { key: 'description', label: 'Description' },
  { key: 'actions', label: '' },
]

function openPONPort(ponPort: PONPort) {
  router.push(`/network/pon-ports/${ponPort.id}`)
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Add/Remove PON Port ---

const showPONPortForm = ref(false)

function handlePONPortCreated(ponPort: PONPort) {
  showPONPortForm.value = false
  ponPorts.value = [...ponPorts.value, ponPort]
}

const ponPortDeleteTarget = ref<PONPort | null>(null)
const ponPortDeletePending = ref(false)
const ponPortDeleteError = ref<string | null>(null)

async function confirmDeletePONPort() {
  const target = ponPortDeleteTarget.value
  if (!target) return
  ponPortDeletePending.value = true
  ponPortDeleteError.value = null
  try {
    await deletePONPort(target.id)
    ponPorts.value = ponPorts.value.filter((ponPort) => ponPort.id !== target.id)
    ponPortDeleteTarget.value = null
  } catch (err) {
    ponPortDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This PON port still has access interfaces attached — remove those first.'
        : 'The PON port could not be deleted.'
  } finally {
    ponPortDeletePending.value = false
  }
}

// --- Edit OLT ---

const showEditDialog = ref(false)

function handleOLTUpdated(updated: OLT) {
  olt.value = updated
  showEditDialog.value = false
}

// --- Delete OLT ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteOLT() {
  if (!olt.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteOLT(olt.value.id)
    router.push(accessNetwork.value ? `/network/${accessNetwork.value.id}` : '/network')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This OLT still has PON ports attached — remove those first.'
        : 'The OLT could not be deleted.'
  } finally {
    deletePending.value = false
  }
}

// --- ONU Status ---

interface ONUStatusResult {
  label: string
  output: string | null
  error: string | null
}

const ONU_STATUS_COMMANDS: { label: string; run: (oltId: string) => Promise<string> }[] = [
  { label: 'ONU Summary', run: runONUSummary },
  { label: 'ONU Status Summary', run: runONUStatusSummary },
]

const onuStatusPending = ref(false)
const onuStatusResults = ref<ONUStatusResult[] | null>(null)

async function checkONUStatus() {
  if (!olt.value) return
  onuStatusPending.value = true

  const results: ONUStatusResult[] = []
  for (const command of ONU_STATUS_COMMANDS) {
    try {
      const output = await command.run(olt.value.id)
      results.push({ label: command.label, output, error: null })
    } catch (err) {
      results.push({ label: command.label, output: null, error: err instanceof ApiError ? err.message : 'This command failed to run.' })
    }
  }

  onuStatusResults.value = results
  onuStatusPending.value = false
}
</script>

<template>
  <div v-if="loading" class="olt-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="olt-detail-view__status">
    <BaseErrorState title="OLT not found" description="This OLT may have been removed, or the link may be out of date.">
      <BaseButton variant="secondary" @click="router.push('/network')">Back to Network</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="olt">
    <WorkspaceHeader :title="olt.name" :subtitle="`${olt.vendor} OLT`" :metadata="headerMetadata">
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit OLT</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete OLT</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <OLTFormDialog
      :open="showEditDialog"
      :access-network-id="olt.accessNetworkId"
      :olt="olt"
      @close="showEditDialog = false"
      @updated="handleOLTUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete OLT"
      :description="`Permanently delete ${olt.name}? This cannot be undone.`"
      confirm-label="Delete OLT"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteOLT"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="network">
      <FactGrid :facts="summaryFacts" />
      <p v-if="olt.description" class="olt-description">{{ olt.description }}</p>
    </SectionCard>

    <SectionCard title="Access Network" icon="network">
      <RelationshipCard
        v-if="accessNetwork"
        eyebrow="Access Network"
        :title="accessNetwork.name"
        :meta="accessNetwork.status"
        :to="`/network/${accessNetwork.id}`"
        action-label="View Access Network"
      />
      <p v-else class="no-relationship">No access network on file for this OLT.</p>
    </SectionCard>

    <SectionCard title="PON Ports" icon="network" :badge="ponPorts.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showPONPortForm = true">Add PON Port</BaseButton>
      </div>

      <PONPortFormDialog
        :open="showPONPortForm"
        :olt-id="olt.id"
        @close="showPONPortForm = false"
        @created="handlePONPortCreated"
      />

      <ConfirmationDialog
        :open="ponPortDeleteTarget !== null"
        title="Remove PON Port"
        :description="`Remove Port ${ponPortDeleteTarget?.portNumber}? This cannot be undone.`"
        confirm-label="Remove PON Port"
        destructive
        :pending="ponPortDeletePending"
        :error="ponPortDeleteError"
        @confirm="confirmDeletePONPort"
        @cancel="ponPortDeleteTarget = null"
      />

      <SimpleTable
        :columns="ponPortColumns"
        :rows="ponPorts"
        :row-key="(ponPort) => ponPort.id"
        clickable
        empty-icon="network"
        empty-title="No PON ports on file"
        @row-click="openPONPort"
      >
        <template #cell-port="{ row }">Port {{ row.portNumber }}</template>
        <template #cell-description="{ row }">{{ row.description || '—' }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="ghost" size="sm" @click.stop="ponPortDeleteTarget = row">Remove</BaseButton>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="ONU Status" icon="network">
      <div class="section-toolbar">
        <BaseButton
          variant="secondary"
          size="sm"
          :disabled="onuStatusPending"
          :disabled-reason="onuStatusPending ? 'Running…' : undefined"
          @click="checkONUStatus"
        >
          {{ onuStatusPending ? 'Checking…' : 'Check ONU Status' }}
        </BaseButton>
      </div>

      <p v-if="!onuStatusResults" class="no-relationship">
        Run a live, on-demand snapshot of every ONU on this OLT.
      </p>

      <div v-else class="onu-status-results">
        <div v-for="result in onuStatusResults" :key="result.label" class="onu-status-result">
          <h4 class="onu-status-result__label">{{ result.label }}</h4>
          <p v-if="result.error" class="onu-status-result__error" role="alert">{{ result.error }}</p>
          <pre v-else class="onu-status-result__output">{{ result.output }}</pre>
        </div>
      </div>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.olt-detail-view__status {
  padding: var(--space-6);
}

.olt-description {
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

.onu-status-results {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.onu-status-result__label {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.onu-status-result__output {
  margin: 0;
  padding: var(--space-3);
  background-color: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  white-space: pre-wrap;
  word-break: break-word;
  overflow-x: auto;
  color: var(--color-text-primary);
}

.onu-status-result__error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
