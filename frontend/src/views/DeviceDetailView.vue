<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import ActivityList from '@/components/data-display/ActivityList.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import NotesList from '@/components/data-display/NotesList.vue'
import RelationshipCard from '@/components/data-display/RelationshipCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import { getDeviceById } from '@/services/devices/deviceRepository'
import { getCustomerById } from '@/services/customers/customerRepository'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { Device, DeviceStatus } from '@/types/device'
import type { Customer, CustomerStatus, ServiceStatus } from '@/types/customer'

/**
 * The Device Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 10, "Device Workspace"): an operational dossier answering
 * "what is this device doing right now?" -- not a configuration form.
 * Every section is read-only this milestone -- no editing, provisioning,
 * or destructive actions, same treatment as the Customer Detail
 * Workspace's header (disabled primary actions with a reason,
 * docs/08-DESIGN-SYSTEM.md section 12).
 *
 * Devices remain projections of Customer -> Service -> Equipment
 * (docs/03-DOMAIN-MODEL.md): the device fetch resolves the device
 * itself, and Assignment additionally resolves the owning customer *on
 * demand* via `assignedCustomerId` (only when present) rather than
 * carrying the full Customer/Service objects on Device permanently --
 * that would duplicate data DeviceCollectionView never needs. Full
 * service detail (`relatedService`) is read out of that fetched
 * customer's own `services` array by `serviceId`, never stored
 * redundantly on Device either.
 */
const route = useRoute()
const router = useRouter()

const device = ref<Device | null>(null)
const relatedCustomer = ref<Customer | null>(null)
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  device.value = null
  relatedCustomer.value = null

  const result = await getDeviceById(id)
  if (result) {
    device.value = result
    if (result.assignedCustomerId) {
      relatedCustomer.value = await getCustomerById(result.assignedCustomerId)
    }
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

const relatedService = computed(() => relatedCustomer.value?.services.find((service) => service.id === device.value?.serviceId))

const STATUS_LABELS: Record<DeviceStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  provisioning: 'Provisioning',
}

const STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info'> = {
  online: 'success',
  offline: 'error',
  warning: 'warning',
  provisioning: 'info',
}

const CUSTOMER_STATUS_LABELS: Record<CustomerStatus, string> = {
  active: 'Active',
  suspended: 'Suspended',
  pending: 'Pending',
  cancelled: 'Cancelled',
}

const SERVICE_STATUS_LABELS: Record<ServiceStatus, string> = {
  active: 'Active',
  provisioning: 'Provisioning',
  suspended: 'Suspended',
  cancelled: 'Cancelled',
}

function managementStateLabel(status: DeviceStatus): string {
  return status === 'provisioning' ? 'Pending Discovery' : 'Managed'
}

function operationalStateLabel(status: DeviceStatus): string {
  switch (status) {
    case 'online':
      return 'Operational'
    case 'warning':
      return 'Degraded'
    case 'offline':
      return 'Unreachable'
    case 'provisioning':
      return 'Provisioning'
  }
}

function provisioningStatusLabel(status: DeviceStatus): string {
  if (status === 'provisioning') return 'Pending Activation'
  if (status === 'offline') return 'Provisioned (Unreachable)'
  return 'Provisioned'
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${minutes}m`
}


const summaryFacts = computed<Fact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: Fact[] = [{ icon: 'devices', label: 'Device Type', value: d.type }]
  if (d.technology) {
    facts.push({ icon: 'network', label: 'Technology', value: d.technology === 'gpon' ? 'GPON' : 'XGS-PON' })
  }
  facts.push(
    { icon: 'clock', label: 'Installed', value: formatDate(d.installedDate) },
    { icon: 'tasks', label: 'Firmware Version', value: d.firmwareVersion },
    { icon: 'health', label: 'Management State', value: managementStateLabel(d.status) },
    { icon: 'inventory', label: 'Vendor', value: d.vendor },
  )
  return facts
})

const networkFacts = computed<Fact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: Fact[] = [{ icon: 'location', label: 'Site', value: d.siteName }]
  if (d.oltId) facts.push({ icon: 'network', label: 'OLT', value: d.oltId })
  if (d.ponPort) facts.push({ icon: 'network', label: 'PON Port', value: d.ponPort })
  if (d.managementIp) facts.push({ icon: 'network', label: 'Management IP', value: d.managementIp })
  if (d.uplinkPort) facts.push({ icon: 'network', label: 'Uplink Port', value: d.uplinkPort })
  return facts
})

const statusFacts = computed<Fact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: Fact[] = [
    { icon: 'health', label: 'Operational State', value: operationalStateLabel(d.status) },
    { icon: 'clock', label: 'Last Contact', value: d.lastContact },
  ]
  if (d.uptimeSeconds !== undefined) {
    facts.push({ icon: 'clock', label: 'Uptime', value: formatUptime(d.uptimeSeconds) })
  }
  if (d.opticalPowerDbm !== undefined) {
    facts.push({ icon: 'network', label: 'Optical Power', value: `${d.opticalPowerDbm} dBm` })
  }
  facts.push(
    { icon: 'alert', label: 'Temperature', value: `${d.temperatureC}°C` },
    { icon: 'tasks', label: 'Provisioning Status', value: provisioningStatusLabel(d.status) },
  )
  return facts
})

const configFacts = computed<Fact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: Fact[] = [{ icon: 'tasks', label: 'Provisioning Profile', value: d.configProfile }]
  if (d.serviceVlan !== undefined) facts.push({ icon: 'network', label: 'Service VLAN', value: String(d.serviceVlan) })
  if (d.managementVlan !== undefined) {
    facts.push({ icon: 'network', label: 'Management VLAN', value: String(d.managementVlan) })
  }
  facts.push({ icon: 'inventory', label: 'Configuration Version', value: d.configVersion })
  return facts
})
</script>

<template>
  <div v-if="loading" class="device-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="device-detail-view__status">
    <BaseErrorState
      title="Device not found"
      description="This device may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/devices')">Back to Devices</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="device">
    <WorkspaceHeader
      :title="device.model"
      :subtitle="device.type"
      :status="{ label: STATUS_LABELS[device.status], variant: STATUS_VARIANTS[device.status] }"
      :metadata="[`Serial ${device.serialNumber}`, device.location]"
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
              Reboot Device
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SectionCard title="Summary" icon="devices">
      <FactGrid :facts="summaryFacts" />
    </SectionCard>

    <SectionCard title="Network" icon="network">
      <FactGrid :facts="networkFacts" />
    </SectionCard>

    <SectionCard title="Assignment" icon="customers">
      <BaseEmptyState
        v-if="!device.assignedCustomerId"
        icon="devices"
        title="Not assigned to a customer"
        description="This is network infrastructure equipment -- it serves many customers rather than belonging to one."
      />
      <div v-else class="assignment-cards">
        <RelationshipCard
          eyebrow="Assigned Customer"
          :title="device.assignedCustomerName ?? 'Customer'"
          :meta="
            relatedCustomer
              ? `${relatedCustomer.type === 'business' ? 'Business' : 'Residential'} · ${CUSTOMER_STATUS_LABELS[relatedCustomer.status]}`
              : undefined
          "
          :to="`/customers/${device.assignedCustomerId}`"
          action-label="View Customer"
        />

        <RelationshipCard
          eyebrow="Assigned Service"
          :title="relatedService?.tier ?? 'Service'"
          :meta="
            relatedService
              ? `${relatedService.technology === 'gpon' ? 'GPON' : 'XGS-PON'} · ${SERVICE_STATUS_LABELS[relatedService.status]}`
              : undefined
          "
          :to="device.serviceId ? `/services/${device.serviceId}` : undefined"
          action-label="View Service"
        />
      </div>
    </SectionCard>

    <SectionCard title="Status" icon="health">
      <FactGrid :facts="statusFacts" />
    </SectionCard>

    <SectionCard title="Configuration">
      <FactGrid :facts="configFacts" />
    </SectionCard>

    <SectionCard title="Recent Activity" icon="clock">
      <ActivityList :entries="device.activity" />
    </SectionCard>

    <SectionCard title="Timeline">
      <TimelineEntries :entries="device.timeline" />
    </SectionCard>

    <SectionCard title="Notes">
      <NotesList :notes="device.notes" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.device-detail-view__status {
  padding: var(--space-6);
}

.assignment-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--space-4);
}
</style>
