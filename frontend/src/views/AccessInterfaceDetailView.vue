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
import AttachAccessAttachmentDialog from '@/components/dialogs/AttachAccessAttachmentDialog.vue'
import DetachAccessAttachmentDialog from '@/components/dialogs/DetachAccessAttachmentDialog.vue'
import { getAccessInterfaceById, deleteAccessInterface } from '@/services/accessInterfaces/accessInterfaceRepository'
import { getPONPortById } from '@/services/ponPorts/ponPortRepository'
import { listAccessAttachmentsByAccessInterfaceId, deleteAccessAttachment } from '@/services/accessAttachments/accessAttachmentRepository'
import { listEvents } from '@/services/events/eventRepository'
import {
  runONUDetail,
  runONURunningConfig,
  runONUStatus,
  runONUEthernetPorts,
  runDHCPSnoopingEntries,
  runMACAddressTableEntries,
} from '@/services/diagnostics/diagnosticsRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { AccessInterface } from '@/types/accessInterface'
import type { PONPort } from '@/types/ponPort'
import type { AccessAttachment } from '@/types/accessAttachment'
import type { TimelineEvent } from '@/types/timelineEvent'

/**
 * The Access Interface Detail Workspace. Same shape as
 * PONPortDetailView.vue/OLTDetailView.vue -- single-relation PON Port
 * section, delete-with-conflict-handling. The Attachments section is
 * this workspace's actual point: attach/detach ServiceEquipment, not
 * create/delete -- see accessAttachmentRepository.ts's own doc comment
 * on why detach is a PUT, not a DELETE. History is not hidden: both
 * active and removed attachments are shown, with a State column
 * distinguishing them.
 *
 * The ONU Status section runs the same per-interface Kontron commands
 * as CustomerDetailView.vue's ONU Diagnostics, plus ONUDetail (the one
 * command that existed end-to-end on the backend but had never been
 * wired into any page) -- but keyed only by this interface's own OLT
 * (ponPort.oltId) and name, with no Customer or ServiceEquipment
 * involved. That is deliberate: an ONU can be physically plugged into
 * an interface well before "Assign Equipment" ever happens on a
 * Service, and a tech doing turn-up needs to check it right here, not
 * by first finding whichever Customer it will eventually belong to.
 */
const route = useRoute()
const router = useRouter()

const accessInterface = ref<AccessInterface | null>(null)
const ponPort = ref<PONPort | null>(null)
const attachments = ref<AccessAttachment[]>([])
const timeline = ref<TimelineEvent[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  accessInterface.value = null
  ponPort.value = null
  attachments.value = []
  timeline.value = []
  onuStatusResults.value = null

  const result = await getAccessInterfaceById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  accessInterface.value = result

  const [relatedPONPort, accessInterfaceAttachments, events] = await Promise.all([
    getPONPortById(result.ponPortId),
    listAccessAttachmentsByAccessInterfaceId(id),
    listEvents('access_interface', id),
  ])
  ponPort.value = relatedPONPort
  attachments.value = accessInterfaceAttachments
  timeline.value = events

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const a = accessInterface.value
  if (!a) return []
  return [
    { icon: 'network', label: 'Technology', value: a.technology },
    { icon: 'health', label: 'Status', value: a.status },
    { icon: 'clock', label: 'Created', value: formatDate(a.createdAt) },
  ]
})

const attachmentColumns: SimpleTableColumn[] = [
  { key: 'equipment', label: 'Equipment' },
  { key: 'state', label: 'State' },
  { key: 'installed', label: 'Installed' },
  { key: 'actions', label: '' },
]

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Attach/Detach Equipment ---

const showAttachDialog = ref(false)

function handleAttachmentCreated(attachment: AccessAttachment) {
  showAttachDialog.value = false
  attachments.value = [...attachments.value, attachment]
}

const detachTarget = ref<AccessAttachment | null>(null)

function handleAttachmentDetached(updated: AccessAttachment) {
  attachments.value = attachments.value.map((attachment) => (attachment.id === updated.id ? updated : attachment))
  detachTarget.value = null
}

const deleteAttachmentTarget = ref<AccessAttachment | null>(null)
const deleteAttachmentPending = ref(false)
const deleteAttachmentError = ref<string | null>(null)

async function confirmDeleteAttachment() {
  if (!deleteAttachmentTarget.value) return
  deleteAttachmentPending.value = true
  deleteAttachmentError.value = null
  try {
    await deleteAccessAttachment(deleteAttachmentTarget.value.id)
    attachments.value = attachments.value.filter((attachment) => attachment.id !== deleteAttachmentTarget.value?.id)
    deleteAttachmentTarget.value = null
  } catch {
    deleteAttachmentError.value = 'This attachment could not be deleted.'
  } finally {
    deleteAttachmentPending.value = false
  }
}

// --- ONU Status ---

interface ONUStatusResult {
  label: string
  output: string | null
  error: string | null
}

const ONU_STATUS_COMMANDS: { label: string; run: (oltId: string, iface: string) => Promise<string> }[] = [
  { label: 'ONU Detail', run: runONUDetail },
  { label: 'Running Configuration', run: runONURunningConfig },
  { label: 'Status', run: runONUStatus },
  { label: 'Ethernet Ports', run: runONUEthernetPorts },
  { label: 'DHCP Snooping', run: runDHCPSnoopingEntries },
  { label: 'MAC Address Table', run: runMACAddressTableEntries },
]

const onuStatusPending = ref(false)
const onuStatusResults = ref<ONUStatusResult[] | null>(null)

async function checkONUStatus() {
  if (!ponPort.value || !accessInterface.value) return
  const oltId = ponPort.value.oltId
  const iface = accessInterface.value.name
  onuStatusPending.value = true

  const results: ONUStatusResult[] = []
  for (const command of ONU_STATUS_COMMANDS) {
    try {
      const output = await command.run(oltId, iface)
      results.push({ label: command.label, output, error: null })
    } catch (err) {
      results.push({ label: command.label, output: null, error: err instanceof ApiError ? err.message : 'This command failed to run.' })
    }
  }

  onuStatusResults.value = results
  onuStatusPending.value = false
}

// --- Edit Access Interface ---

const showEditDialog = ref(false)

function handleAccessInterfaceUpdated(updated: AccessInterface) {
  accessInterface.value = updated
  showEditDialog.value = false
}

// --- Delete Access Interface ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteAccessInterface() {
  if (!accessInterface.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteAccessInterface(accessInterface.value.id)
    router.push(ponPort.value ? `/network/pon-ports/${ponPort.value.id}` : '/network')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This access interface still has attachments — remove those first.'
        : 'The access interface could not be deleted.'
  } finally {
    deletePending.value = false
  }
}
</script>

<template>
  <div v-if="loading" class="access-interface-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="access-interface-detail-view__status">
    <BaseErrorState
      title="Access interface not found"
      description="This access interface may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/network')">Back to Network</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="accessInterface">
    <WorkspaceHeader
      :title="accessInterface.name"
      :subtitle="accessInterface.technology"
      :status="{ label: accessInterface.status, variant: accessInterface.status === 'Active' ? 'success' : 'neutral' }"
      :metadata="[`Access Interface ${accessInterface.id}`]"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="secondary" size="sm" @click="showEditDialog = true">Edit Access Interface</BaseButton>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">
              Delete Access Interface
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <AccessInterfaceFormDialog
      :open="showEditDialog"
      :pon-port-id="accessInterface.ponPortId"
      :access-interface="accessInterface"
      @close="showEditDialog = false"
      @updated="handleAccessInterfaceUpdated"
    />

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Access Interface"
      :description="`Permanently delete ${accessInterface.name}? This cannot be undone.`"
      confirm-label="Delete Access Interface"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteAccessInterface"
      @cancel="showDeleteDialog = false"
    />

    <SectionCard title="Summary" icon="network">
      <FactGrid :facts="summaryFacts" />
      <p v-if="accessInterface.description" class="access-interface-description">{{ accessInterface.description }}</p>
    </SectionCard>

    <SectionCard title="PON Port" icon="network">
      <RelationshipCard
        v-if="ponPort"
        eyebrow="PON Port"
        :title="`Port ${ponPort.portNumber}`"
        :to="`/network/pon-ports/${ponPort.id}`"
        action-label="View PON Port"
      />
      <p v-else class="no-relationship">No PON port on file for this access interface.</p>
    </SectionCard>

    <SectionCard title="Attachments" icon="services" :badge="attachments.length">
      <div class="section-toolbar">
        <BaseButton variant="secondary" size="sm" @click="showAttachDialog = true">Attach Equipment</BaseButton>
      </div>

      <AttachAccessAttachmentDialog
        :open="showAttachDialog"
        :access-interface-id="accessInterface.id"
        @close="showAttachDialog = false"
        @created="handleAttachmentCreated"
      />

      <DetachAccessAttachmentDialog
        v-if="detachTarget"
        :open="true"
        :attachment="detachTarget"
        @close="detachTarget = null"
        @detached="handleAttachmentDetached"
      />

      <ConfirmationDialog
        :open="deleteAttachmentTarget !== null"
        title="Delete Attachment"
        description="Permanently delete this attachment record? This cannot be undone."
        confirm-label="Delete Attachment"
        destructive
        :pending="deleteAttachmentPending"
        :error="deleteAttachmentError"
        @confirm="confirmDeleteAttachment"
        @cancel="deleteAttachmentTarget = null"
      />

      <SimpleTable
        :columns="attachmentColumns"
        :rows="attachments"
        :row-key="(attachment) => attachment.id"
        empty-icon="services"
        empty-title="No equipment attached"
      >
        <template #cell-equipment="{ row }"><span class="cell-mono">{{ row.serviceEquipmentId }}</span></template>
        <template #cell-state="{ row }">{{ row.removedAt === null ? 'Active' : 'Removed' }}</template>
        <template #cell-installed="{ row }">{{ row.installedAt ? formatDate(row.installedAt) : 'Not yet installed' }}</template>
        <template #cell-actions="{ row }">
          <BaseButton v-if="row.removedAt === null" variant="ghost" size="sm" @click="detachTarget = row">Detach</BaseButton>
          <BaseButton v-else variant="ghost" size="sm" @click="deleteAttachmentTarget = row">Delete</BaseButton>
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
        Run a live, on-demand check of whatever ONU is plugged into this interface right now.
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
.access-interface-detail-view__status {
  padding: var(--space-6);
}

.access-interface-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.cell-mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
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
