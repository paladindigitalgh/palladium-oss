<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import ActivityList from '@/components/data-display/ActivityList.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'
import NotesList from '@/components/data-display/NotesList.vue'
import FactGrid, { type Fact } from '@/components/data-display/FactGrid.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import { getCustomerById } from '@/services/customers/customerRepository'
import type {
  AlertSeverity,
  AssetStatus,
  Customer,
  CustomerAsset,
  CustomerService,
  CustomerStatus,
  ServiceStatus,
} from '@/types/customer'

/**
 * The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace"): an operational dossier, not a form.
 * Every section is read-only this milestone -- no editing, provisioning,
 * or destructive actions (docs/02-DESIGN-PRINCIPLES.md principle 9,
 * "Read Before Write," read literally: understand first). The header's
 * primary actions are rendered disabled with a reason
 * (docs/08-DESIGN-SYSTEM.md section 12: "Disable actions only when
 * necessary and explain why") rather than omitted entirely, since the
 * header still needs to establish where those actions will eventually
 * live once the Workflow Engine exists.
 *
 * Devices are not a customer-owned list in the data model
 * (docs/03-DOMAIN-MODEL.md section 4) -- they are read off each
 * service's own equipment and flattened for display, matching
 * "Correlation Over Collection" (docs/02-DESIGN-PRINCIPLES.md principle
 * 11).
 */
const route = useRoute()
const router = useRouter()

const customer = ref<Customer | null>(null)
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  customer.value = null

  const result = await getCustomerById(id)
  if (result) {
    customer.value = result
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

const STATUS_LABELS: Record<CustomerStatus, string> = {
  active: 'Active',
  suspended: 'Suspended',
  pending: 'Pending',
  cancelled: 'Cancelled',
}

const STATUS_VARIANTS: Record<CustomerStatus, 'success' | 'warning' | 'error'> = {
  active: 'success',
  pending: 'warning',
  suspended: 'warning',
  cancelled: 'error',
}

const SERVICE_STATUS_LABELS: Record<ServiceStatus, string> = {
  active: 'Active',
  suspended: 'Suspended',
  pending: 'Pending',
  decommissioned: 'Decommissioned',
}

const SERVICE_STATUS_VARIANTS: Record<ServiceStatus, 'success' | 'warning' | 'error' | 'neutral'> = {
  active: 'success',
  suspended: 'warning',
  pending: 'warning',
  decommissioned: 'neutral',
}

const ASSET_STATUS_LABELS: Record<AssetStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  unknown: 'Unknown',
}

const ASSET_STATUS_VARIANTS: Record<AssetStatus, 'success' | 'error' | 'neutral'> = {
  online: 'success',
  offline: 'error',
  unknown: 'neutral',
}

const ALERT_SEVERITY_LABELS: Record<AlertSeverity, string> = {
  critical: 'Critical',
  warning: 'Warning',
  info: 'Info',
}

const ALERT_SEVERITY_VARIANTS: Record<AlertSeverity, 'error' | 'warning' | 'info'> = {
  critical: 'error',
  warning: 'warning',
  info: 'info',
}

function formatDate(iso: string): string {
  const [year, month, day] = iso.split('-').map(Number)
  return new Date(year, month - 1, day).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

const summaryFacts = computed<Fact[]>(() => {
  const c = customer.value
  if (!c) return []
  const primary = c.services[0]
  return [
    { icon: 'health', label: 'Status', value: STATUS_LABELS[c.status] },
    { icon: 'customers', label: 'Customer Type', value: c.type === 'business' ? 'Business' : 'Residential' },
    { icon: 'services', label: 'Primary Service', value: primary.tier },
    { icon: 'clock', label: 'Provisioned', value: formatDate(c.installDate) },
    { icon: 'location', label: 'Service Address', value: primary.serviceAddress },
    { icon: 'user', label: 'Primary Contact', value: `${c.contacts.primary.name} · ${c.contacts.primary.phone}` },
  ]
})

const serviceColumns: SimpleTableColumn[] = [
  { key: 'service', label: 'Service' },
  { key: 'technology', label: 'Technology' },
  { key: 'status', label: 'Status' },
  { key: 'provisioned', label: 'Provisioned' },
]

const deviceColumns: SimpleTableColumn[] = [
  { key: 'device', label: 'Device' },
  { key: 'role', label: 'Role' },
  { key: 'status', label: 'Status' },
  { key: 'serial', label: 'Serial' },
]

const devices = computed<CustomerAsset[]>(() => customer.value?.services.flatMap((service) => service.equipment) ?? [])

function serviceRowKey(service: CustomerService): string {
  return service.id
}

function deviceRowKey(device: CustomerAsset): string {
  return device.id
}

function deviceRowLabel(device: CustomerAsset): string {
  return `Open ${device.model}`
}

function openDevice(device: CustomerAsset) {
  router.push(`/devices/${device.id}`)
}
</script>

<template>
  <div v-if="loading" class="customer-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="customer-detail-view__status">
    <BaseErrorState
      title="Customer not found"
      description="This customer may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/customers')">Back to Customers</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="customer">
    <WorkspaceHeader
      :title="customer.name"
      :subtitle="customer.type === 'business' ? 'Business Customer' : 'Residential Customer'"
      :status="{ label: STATUS_LABELS[customer.status], variant: STATUS_VARIANTS[customer.status] }"
      :metadata="[`Customer #${customer.id}`, customer.services[0].serviceAddress]"
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
              Launch Diagnostics
            </BaseButton>
          </template>
          <template #primary>
            <BaseButton
              variant="primary"
              size="sm"
              disabled
              disabled-reason="Workflow actions are not yet implemented."
            >
              Provision Service
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SectionCard title="Summary" icon="customers">
      <FactGrid :facts="summaryFacts" />
    </SectionCard>

    <SectionCard title="Services" icon="services" :badge="customer.services.length">
      <SimpleTable
        :columns="serviceColumns"
        :rows="customer.services"
        :row-key="serviceRowKey"
        empty-icon="services"
        empty-title="No services on this account"
      >
        <template #cell-service="{ row }">
          <span class="cell-strong">{{ row.tier }}</span>
        </template>
        <template #cell-technology="{ row }">
          <BaseBadge variant="info">{{ row.technology === 'gpon' ? 'GPON' : 'XGS-PON' }}</BaseBadge>
        </template>
        <template #cell-status="{ row }">
          <BaseBadge :variant="SERVICE_STATUS_VARIANTS[row.status as ServiceStatus]">{{
            SERVICE_STATUS_LABELS[row.status as ServiceStatus]
          }}</BaseBadge>
        </template>
        <template #cell-provisioned="{ row }">
          {{ formatDate(row.provisionedDate) }}
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Devices" icon="devices" :badge="devices.length">
      <SimpleTable
        :columns="deviceColumns"
        :rows="devices"
        :row-key="deviceRowKey"
        :row-label="deviceRowLabel"
        clickable
        empty-icon="devices"
        empty-title="No equipment assigned"
        @row-click="openDevice"
      >
        <template #cell-device="{ row }">
          <span class="cell-strong">{{ row.model }}</span>
        </template>
        <template #cell-role="{ row }">
          {{ row.role }}
        </template>
        <template #cell-status="{ row }">
          <BaseBadge :variant="ASSET_STATUS_VARIANTS[row.status as AssetStatus]">{{
            ASSET_STATUS_LABELS[row.status as AssetStatus]
          }}</BaseBadge>
        </template>
        <template #cell-serial="{ row }">
          <span class="cell-mono">{{ row.serialNumber }}</span>
        </template>
      </SimpleTable>
    </SectionCard>

    <SectionCard title="Contacts" icon="user">
      <div class="contacts">
        <div class="contact">
          <span class="contact__eyebrow">Primary Contact</span>
          <p class="contact__name">
            {{ customer.contacts.primary.name }}
            <span v-if="customer.contacts.primary.role" class="contact__role"
              >({{ customer.contacts.primary.role }})</span
            >
          </p>
          <a class="contact__link" :href="`tel:${customer.contacts.primary.phone}`">{{
            customer.contacts.primary.phone
          }}</a>
          <a class="contact__link" :href="`mailto:${customer.contacts.primary.email}`">{{
            customer.contacts.primary.email
          }}</a>
        </div>
        <div v-if="customer.contacts.secondary" class="contact">
          <span class="contact__eyebrow">Secondary Contact</span>
          <p class="contact__name">
            {{ customer.contacts.secondary.name }}
            <span v-if="customer.contacts.secondary.role" class="contact__role"
              >({{ customer.contacts.secondary.role }})</span
            >
          </p>
          <a class="contact__link" :href="`tel:${customer.contacts.secondary.phone}`">{{
            customer.contacts.secondary.phone
          }}</a>
          <a class="contact__link" :href="`mailto:${customer.contacts.secondary.email}`">{{
            customer.contacts.secondary.email
          }}</a>
        </div>
      </div>
    </SectionCard>

    <SectionCard title="Alerts" icon="alert" :badge="customer.alerts.length || undefined">
      <BaseEmptyState
        v-if="customer.alerts.length === 0"
        icon="check"
        title="No active alerts"
        description="This customer has no active alerts."
      />
      <ul v-else class="alert-list">
        <li v-for="alert in customer.alerts" :key="alert.id" class="alert-list__item">
          <BaseBadge :variant="ALERT_SEVERITY_VARIANTS[alert.severity]">{{
            ALERT_SEVERITY_LABELS[alert.severity]
          }}</BaseBadge>
          <div class="alert-list__body">
            <p class="alert-list__title">{{ alert.title }}</p>
            <p class="alert-list__description">{{ alert.description }}</p>
          </div>
          <span class="alert-list__timestamp">{{ alert.timestamp }}</span>
        </li>
      </ul>
    </SectionCard>

    <SectionCard title="Recent Activity" icon="clock">
      <ActivityList :entries="customer.activity" />
    </SectionCard>

    <SectionCard title="Timeline">
      <TimelineEntries :entries="customer.timeline" />
    </SectionCard>

    <SectionCard title="Notes">
      <NotesList :notes="customer.notes" />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.customer-detail-view__status {
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

.contacts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-5);
}

.contact {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.contact__eyebrow {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: var(--space-1);
}

.contact__name {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.contact__role {
  font-weight: var(--font-weight-regular);
  color: var(--color-text-muted);
}

.contact__link {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  text-decoration: none;
  width: fit-content;
}

.contact__link:hover {
  color: var(--color-brand);
  text-decoration: underline;
}

.alert-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.alert-list__item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.alert-list__body {
  flex: 1;
  min-width: 0;
}

.alert-list__title {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.alert-list__description {
  margin-top: 2px;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.alert-list__timestamp {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
}

</style>
