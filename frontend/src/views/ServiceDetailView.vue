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
import ActivityList from '@/components/data-display/ActivityList.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import NotesList from '@/components/data-display/NotesList.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import { getServiceById } from '@/services/services/serviceRepository'
import { getCustomerById } from '@/services/customers/customerRepository'
import { listDevicesByServiceId } from '@/services/devices/deviceRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { Service, ServiceCategory } from '@/types/service'
import type { Customer, CustomerStatus, ServiceStatus } from '@/types/customer'
import type { Device, DeviceStatus } from '@/types/device'

/**
 * The Service Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 9, "Service Workspace"): answers "what is being delivered?",
 * distinct from Customer ("who receives service?") and Device ("what
 * equipment exists?"). Read-only this milestone -- no editing,
 * provisioning, or destructive actions, same header-actions treatment as
 * Customer/Device Detail (disabled with a reason).
 *
 * Completes the relationship triangle: the Customer card and every
 * Devices row navigate to their own canonical Detail Workspace, and
 * Customer Detail's Services rows / Device Detail's Assignment card both
 * now navigate here in turn. Neither the related customer nor the
 * related devices are stored on Service itself (services/services/
 * serviceDataset.ts never carries them) -- both are resolved on demand
 * once the service is known, the same pattern Device Detail already
 * uses for its own Assignment section.
 */
const route = useRoute()
const router = useRouter()

const service = ref<Service | null>(null)
const relatedCustomer = ref<Customer | null>(null)
const relatedDevices = ref<Device[]>([])
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  service.value = null
  relatedCustomer.value = null
  relatedDevices.value = []

  const result = await getServiceById(id)
  if (result) {
    service.value = result
    const [customer, devices] = await Promise.all([
      getCustomerById(result.customerId),
      listDevicesByServiceId(result.id),
    ])
    relatedCustomer.value = customer
    relatedDevices.value = devices
  } else {
    notFound.value = true
  }
  loading.value = false
}

onMounted(() => load(route.params.id as string))
watch(
  () => route.params.id,
  (id) => load(id as string),
)

const STATUS_LABELS: Record<ServiceStatus, string> = {
  active: 'Active',
  provisioning: 'Provisioning',
  suspended: 'Suspended',
  cancelled: 'Cancelled',
}

const STATUS_VARIANTS: Record<ServiceStatus, 'success' | 'info' | 'warning' | 'error'> = {
  active: 'success',
  provisioning: 'info',
  suspended: 'warning',
  cancelled: 'error',
}

const CATEGORY_LABELS: Record<ServiceCategory, string> = {
  internet: 'Internet',
  'internet-static-ipv4': 'Internet + Static IPv4',
  'internet-ipv6': 'Internet + IPv6',
  'business-internet': 'Business Internet',
}

const CUSTOMER_STATUS_LABELS: Record<CustomerStatus, string> = {
  active: 'Active',
  suspended: 'Suspended',
  pending: 'Pending',
  cancelled: 'Cancelled',
}

const DEVICE_STATUS_LABELS: Record<DeviceStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  provisioning: 'Provisioning',
}

const DEVICE_STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info'> = {
  online: 'success',
  offline: 'error',
  warning: 'warning',
  provisioning: 'info',
}

const DEVICE_STATUS_RANK: Record<DeviceStatus, number> = { offline: 0, warning: 1, provisioning: 2, online: 3 }
const DEVICE_OPERATIONAL_LABELS: Record<DeviceStatus, string> = {
  online: 'Operational',
  warning: 'Degraded',
  offline: 'Unreachable',
  provisioning: 'Provisioning',
}

function provisioningStateLabel(status: ServiceStatus): string {
  switch (status) {
    case 'provisioning':
      return 'Pending Activation'
    case 'active':
      return 'Fully Provisioned'
    case 'suspended':
      return 'Provisioned (Suspended)'
    case 'cancelled':
      return 'Deprovisioned'
  }
}

const operationalState = computed(() => {
  if (relatedDevices.value.length === 0) return 'Unknown'
  const worst = relatedDevices.value.reduce((worst, device) =>
    DEVICE_STATUS_RANK[device.status] < DEVICE_STATUS_RANK[worst.status] ? device : worst,
  )
  return DEVICE_OPERATIONAL_LABELS[worst.status]
})

const summaryFacts = computed<Fact[]>(() => {
  const s = service.value
  if (!s) return []
  return [
    { icon: 'network', label: 'Technology', value: s.technology === 'gpon' ? 'GPON' : 'XGS-PON' },
    { icon: 'services', label: 'Service Type', value: CATEGORY_LABELS[s.category] },
    { icon: 'clock', label: 'Provisioned', value: formatDate(s.provisionedDate) },
    { icon: 'clock', label: 'Activation Date', value: s.activationDate ? formatDate(s.activationDate) : 'Not yet activated' },
    { icon: 'health', label: 'Current Status', value: STATUS_LABELS[s.status] },
  ]
})

const provisioningFacts = computed<Fact[]>(() => {
  const s = service.value
  if (!s) return []
  return [
    { icon: 'tasks', label: 'Provisioning Profile', value: s.provisioningProfile },
    { icon: 'network', label: 'Bandwidth Profile', value: s.bandwidthProfile },
    { icon: 'user', label: 'Authentication Profile', value: s.authenticationProfile },
    { icon: 'inventory', label: 'Configuration Profile', value: s.configurationProfile },
  ]
})

const networkFacts = computed<Fact[]>(() => {
  const s = service.value
  if (!s) return []
  const facts: Fact[] = []
  if (s.oltId) facts.push({ icon: 'network', label: 'OLT', value: s.oltId })
  if (s.ponPort) facts.push({ icon: 'network', label: 'PON Port', value: s.ponPort })
  if (s.serviceVlan !== undefined) facts.push({ icon: 'network', label: 'Service VLAN', value: String(s.serviceVlan) })
  if (s.managementVlan !== undefined) {
    facts.push({ icon: 'network', label: 'Management VLAN', value: String(s.managementVlan) })
  }
  if (s.ipv4Address) facts.push({ icon: 'network', label: 'IPv4', value: s.ipv4Address })
  if (s.ipv6Address) facts.push({ icon: 'network', label: 'IPv6', value: s.ipv6Address })
  if (s.gateway) facts.push({ icon: 'network', label: 'Gateway', value: s.gateway })
  return facts
})

const statusFacts = computed<Fact[]>(() => {
  const s = service.value
  if (!s) return []
  return [
    { icon: 'health', label: 'Operational State', value: operationalState.value },
    { icon: 'tasks', label: 'Provisioning State', value: provisioningStateLabel(s.status) },
    { icon: 'clock', label: 'Last Successful Synchronization', value: s.lastSync },
  ]
})

const deviceColumns: SimpleTableColumn[] = [
  { key: 'device', label: 'Device' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'serial', label: 'Serial' },
]

function deviceRowKey(device: Device): string {
  return device.id
}

function deviceRowLabel(device: Device): string {
  return `Open ${device.model}`
}

function openDevice(device: Device) {
  router.push(`/devices/${device.id}`)
}
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
      :title="service.tier"
      :subtitle="CATEGORY_LABELS[service.category]"
      :status="{ label: STATUS_LABELS[service.status], variant: STATUS_VARIANTS[service.status] }"
      :metadata="[`Service #${service.id}`, service.serviceAddress]"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton
              variant="ghost"
              size="sm"
              disabled
              disabled-reason="Workflow actions are not yet implemented."
            >
              Run Diagnostics
            </BaseButton>
          </template>
          <template #primary>
            <BaseButton
              variant="primary"
              size="sm"
              disabled
              disabled-reason="Workflow actions are not yet implemented."
            >
              Suspend Service
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SectionCard title="Summary" icon="services">
      <FactGrid :facts="summaryFacts" />
    </SectionCard>

    <SectionCard title="Customer" icon="customers">
      <RelationshipCard
        eyebrow="Customer"
        :title="service.customerName"
        :meta="
          relatedCustomer
            ? `${relatedCustomer.type === 'business' ? 'Business' : 'Residential'} · ${CUSTOMER_STATUS_LABELS[relatedCustomer.status]}`
            : undefined
        "
        :to="`/customers/${service.customerId}`"
        action-label="View Customer"
      />
    </SectionCard>

    <SectionCard title="Devices" icon="devices" :badge="relatedDevices.length">
      <SimpleTable
        :columns="deviceColumns"
        :rows="relatedDevices"
        :row-key="deviceRowKey"
        :row-label="deviceRowLabel"
        clickable
        empty-icon="devices"
        empty-title="No devices delivering this service"
        @row-click="openDevice"
      >
        <template #cell-device="{ row }">
          <span class="cell-strong">{{ row.model }}</span>
        </template>
        <template #cell-role="{ row }">
          {{ row.type }}
        </template>
        <template #cell-status="{ row }">
          <BaseBadge :variant="DEVICE_STATUS_VARIANTS[row.status as DeviceStatus]">{{
            DEVICE_STATUS_LABELS[row.status as DeviceStatus]
          }}</BaseBadge>
        </template>
        <template #cell-serial="{ row }">
          <span class="cell-mono">{{ row.serialNumber }}</span>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Provisioning" icon="tasks">
      <FactGrid :facts="provisioningFacts" />
    </SectionCard>

    <SectionCard title="Network" icon="network">
      <FactGrid :facts="networkFacts" />
    </SectionCard>

    <SectionCard title="Status" icon="health">
      <FactGrid :facts="statusFacts" />
    </SectionCard>

    <SectionCard title="Recent Activity" icon="clock">
      <ActivityList :entries="service.activity" />
    </SectionCard>

    <SectionCard title="Timeline">
      <TimelineEntries :entries="service.timeline" />
    </SectionCard>

    <SectionCard title="Notes">
      <NotesList :notes="service.notes" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.service-detail-view__status {
  padding: var(--space-6);
}

.cell-strong {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.cell-mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}
</style>
