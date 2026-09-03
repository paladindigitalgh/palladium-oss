<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import RelationshipCard from '@/components/data-display/RelationshipCard.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import ConfirmationDialog from '@/components/dialogs/ConfirmationDialog.vue'
import { getServiceById, deleteService } from '@/services/services/serviceRepository'
import { getLocationById } from '@/services/locations/locationRepository'
import { getCustomerById } from '@/services/customers/customerRepository'
import { listServiceEquipmentByServiceId } from '@/services/serviceEquipment/serviceEquipmentRepository'
import { getDeviceById } from '@/services/devices/deviceRepository'
import { listEvents } from '@/services/events/eventRepository'
import { listWorkflowInstancesByServiceId, runWorkflow } from '@/services/workflow/workflowRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import { ApiError } from '@/services/api/httpClient'
import type { Service } from '@/types/service'
import type { Location } from '@/types/location'
import type { Customer } from '@/types/customer'
import type { Device } from '@/types/device'
import type { ServiceEquipment } from '@/types/serviceEquipment'
import type { TimelineEvent } from '@/types/timelineEvent'
import type { WorkflowDefinitionName, WorkflowInstance } from '@/types/workflowInstance'

/**
 * The Service Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 9, "Service Workspace"), backed by the real backend -- this is
 * where the Workflow Engine loop is actually exercised: Provision,
 * Suspend, and Resume run a real WorkflowInstance against
 * internal/plugin/mock's simulated vendor (docs/05-WORKFLOW-ENGINE.md),
 * and the Service's own Status updates when it succeeds (see
 * internal/workflow/engine's Execute).
 *
 * Sections that depended on mock-only concepts (Provisioning Profile,
 * Network/VLAN detail) are removed rather than faked. Equipment shows
 * the real, lean Service Equipment assignment, with each row's Device
 * resolved on demand -- one apiFetch per unique deviceId, not embedded
 * on ServiceEquipment itself (see types/serviceEquipment.ts).
 */
const route = useRoute()
const router = useRouter()

const service = ref<Service | null>(null)
const location = ref<Location | null>(null)
const customer = ref<Customer | null>(null)
const equipment = ref<ServiceEquipment[]>([])
const devicesById = ref<Map<string, Device>>(new Map())
const timeline = ref<TimelineEvent[]>([])
const workflowHistory = ref<WorkflowInstance[]>([])
const loading = ref(true)
const notFound = ref(false)
const actionPending = ref(false)
const actionError = ref<string | null>(null)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  actionError.value = null
  service.value = null
  location.value = null
  customer.value = null
  equipment.value = []
  devicesById.value = new Map()
  timeline.value = []
  workflowHistory.value = []

  const result = await getServiceById(id)
  if (!result) {
    notFound.value = true
    loading.value = false
    return
  }
  service.value = result

  const [relatedLocation, relatedEquipment, events, history] = await Promise.all([
    getLocationById(result.locationId),
    listServiceEquipmentByServiceId(result.id),
    listEvents('service', result.id),
    listWorkflowInstancesByServiceId(result.id),
  ])
  location.value = relatedLocation
  equipment.value = relatedEquipment
  timeline.value = events
  workflowHistory.value = history

  if (relatedLocation) {
    customer.value = await getCustomerById(relatedLocation.customerId)
  }

  const uniqueDeviceIds = [...new Set(relatedEquipment.map((item) => item.deviceId))]
  const devices = await Promise.all(uniqueDeviceIds.map((deviceId) => getDeviceById(deviceId)))
  const byId = new Map<string, Device>()
  uniqueDeviceIds.forEach((deviceId, index) => {
    const device = devices[index]
    if (device) byId.set(deviceId, device)
  })
  devicesById.value = byId

  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const summaryFacts = computed<Fact[]>(() => {
  const s = service.value
  if (!s) return []
  const facts: Fact[] = [{ icon: 'health', label: 'Status', value: s.status }]
  if (s.activatedAt) facts.push({ icon: 'clock', label: 'Activated', value: formatDate(s.activatedAt) })
  if (s.suspendedAt) facts.push({ icon: 'clock', label: 'Suspended', value: formatDate(s.suspendedAt) })
  if (s.disconnectedAt) facts.push({ icon: 'clock', label: 'Disconnected', value: formatDate(s.disconnectedAt) })
  facts.push({ icon: 'clock', label: 'Last Updated', value: formatDate(s.updatedAt) })
  return facts
})

const equipmentColumns: SimpleTableColumn[] = [
  { key: 'role', label: 'Role' },
  { key: 'device', label: 'Device' },
  { key: 'installed', label: 'Installed' },
]

/**
 * Which workflow the primary action button runs, derived from the
 * Service's current Status -- exactly what internal/workflow/engine's
 * serviceStatusAfter maps back, so the button offered here always
 * matches a transition the engine will actually accept.
 */
const primaryAction = computed<{ label: string; definition: WorkflowDefinitionName } | null>(() => {
  switch (service.value?.status) {
    case 'Pending':
      return { label: 'Provision Service', definition: 'provision-service' }
    case 'Active':
      return { label: 'Suspend Service', definition: 'suspend-service' }
    case 'Suspended':
      return { label: 'Resume Service', definition: 'resume-service' }
    default:
      return null
  }
})

async function runAction(definition: WorkflowDefinitionName) {
  if (!service.value) return
  actionPending.value = true
  actionError.value = null
  try {
    await runWorkflow(service.value.id, definition)
    await load(service.value.id)
  } catch (err) {
    actionError.value = err instanceof ApiError ? err.message : 'The workflow could not be executed.'
  } finally {
    actionPending.value = false
  }
}

const timelineEntries = computed(() =>
  timeline.value.map((event) => ({ id: event.id, label: event.message, timestamp: event.createdAt, description: event.type })),
)

// --- Delete Service ---

const showDeleteDialog = ref(false)
const deletePending = ref(false)
const deleteError = ref<string | null>(null)

async function confirmDeleteService() {
  if (!service.value) return
  deletePending.value = true
  deleteError.value = null
  try {
    await deleteService(service.value.id)
    router.push(customer.value ? `/customers/${customer.value.id}` : '/services')
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This service still has equipment or workflow history attached — remove those first.'
        : 'The service could not be deleted.'
  } finally {
    deletePending.value = false
  }
}

const workflowColumns: SimpleTableColumn[] = [
  { key: 'definition', label: 'Workflow' },
  { key: 'status', label: 'Status' },
  { key: 'started', label: 'Started' },
]
</script>

<template>
  <div v-if="loading" class="service-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="service-detail-view__status">
    <BaseErrorState
      title="Service not found"
      description="This service may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/services')">Back to Services</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="service">
    <WorkspaceHeader
      :title="`Service ${service.id}`"
      :status="{ label: service.status, variant: service.status === 'Active' ? 'success' : 'neutral' }"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="destructive" size="sm" @click="showDeleteDialog = true">Delete Service</BaseButton>
          </template>
          <template v-if="primaryAction" #primary>
            <BaseButton
              variant="primary"
              size="sm"
              :disabled="actionPending"
              :disabled-reason="actionPending ? 'Running…' : undefined"
              @click="runAction(primaryAction.definition)"
            >
              {{ actionPending ? 'Running…' : primaryAction.label }}
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <ConfirmationDialog
      :open="showDeleteDialog"
      title="Delete Service"
      :description="`Permanently delete service ${service.id}? This cannot be undone.`"
      confirm-label="Delete Service"
      destructive
      :pending="deletePending"
      :error="deleteError"
      @confirm="confirmDeleteService"
      @cancel="showDeleteDialog = false"
    />

    <p v-if="actionError" class="action-error" role="alert">{{ actionError }}</p>

    <SectionCard title="Summary" icon="services">
      <FactGrid :facts="summaryFacts" />
      <p v-if="service.description" class="service-description">{{ service.description }}</p>
    </SectionCard>

    <SectionCard title="Customer" icon="customers">
      <RelationshipCard
        v-if="customer"
        eyebrow="Customer"
        :title="customer.name"
        :meta="`${customer.customerType} · ${customer.status}`"
        :to="`/customers/${customer.id}`"
        action-label="View Customer"
      />
      <p v-else class="no-relationship">No customer on file for this service's location.</p>
    </SectionCard>

    <SectionCard title="Location" icon="location">
      <p v-if="location" class="location-summary">
        {{ location.name }} — {{ location.address1 }}, {{ location.city }}, {{ location.state }} {{ location.postalCode }}
      </p>
      <p v-else class="no-relationship">No location on file for this service.</p>
    </SectionCard>

    <SectionCard title="Equipment" icon="devices" :badge="equipment.length">
      <SimpleTable
        :columns="equipmentColumns"
        :rows="equipment"
        :row-key="(item) => item.id"
        empty-icon="devices"
        empty-title="No equipment assigned"
      >
        <template #cell-role="{ row }">{{ row.role }}</template>
        <template #cell-device="{ row }">
          <span v-if="devicesById.get(row.deviceId)">
            {{ devicesById.get(row.deviceId)!.manufacturer }} {{ devicesById.get(row.deviceId)!.model }} —
            {{ devicesById.get(row.deviceId)!.serialNumber }}
          </span>
          <span v-else class="cell-mono">{{ row.deviceId }}</span>
        </template>
        <template #cell-installed="{ row }">{{ row.installedAt ? formatDate(row.installedAt) : 'Not yet installed' }}</template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Workflow History" icon="tasks" :badge="workflowHistory.length">
      <SimpleTable
        :columns="workflowColumns"
        :rows="workflowHistory"
        :row-key="(instance) => instance.id"
        empty-icon="tasks"
        empty-title="No workflows have run against this service yet"
      >
        <template #cell-definition="{ row }">{{ row.definitionName }}</template>
        <template #cell-status="{ row }">{{ row.status }}</template>
        <template #cell-started="{ row }">{{ row.startedAt ? formatDate(row.startedAt) : '—' }}</template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Timeline" icon="history">
      <TimelineEntries :entries="timelineEntries" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.service-detail-view__status {
  padding: var(--space-6);
}

.cell-mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.service-description {
  margin-top: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.no-relationship {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.location-summary {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.action-error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
