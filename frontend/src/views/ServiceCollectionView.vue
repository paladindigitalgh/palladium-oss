<script setup lang="ts">
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import type { Service } from '@/types/service'
import type { ServiceStatus } from '@/types/customer'
import { useServiceCollection, type ServiceSortKey } from '@/composables/useServiceCollection'

/**
 * The Service Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"): discovery only, for finding what is
 * being delivered -- distinct from Customers ("who receives service")
 * and Devices ("what equipment exists"). Built entirely from the same
 * Collection Workspace components Customers and Devices already
 * established (DataTable, CollectionToolbar, BaseSelect) -- nothing
 * service-specific lives in those components, only in this view and its
 * own composable/repository.
 */
const router = useRouter()

const { search, status, technology, category, customerType, sortKey, sortDirection, toggleSort, page, pageSize, services, total, loading } =
  useServiceCollection()

const columns: DataTableColumn[] = [
  { key: 'service', label: 'Service', sortable: true },
  { key: 'customer', label: 'Customer', sortable: true },
  { key: 'technology', label: 'Technology', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'active', label: 'Active' },
  { value: 'provisioning', label: 'Provisioning' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'cancelled', label: 'Cancelled' },
]

const technologyOptions = [
  { value: 'any', label: 'Any' },
  { value: 'gpon', label: 'GPON' },
  { value: 'xgs-pon', label: 'XGS-PON' },
]

const categoryOptions = [
  { value: 'all', label: 'All Service Types' },
  { value: 'internet', label: 'Internet' },
  { value: 'internet-static-ipv4', label: 'Internet + Static IPv4' },
  { value: 'internet-ipv6', label: 'Internet + IPv6' },
  { value: 'business-internet', label: 'Business Internet' },
]

const customerTypeOptions = [
  { value: 'all', label: 'All Customer Types' },
  { value: 'residential', label: 'Residential' },
  { value: 'business', label: 'Business' },
]

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

function rowLabel(service: Service): string {
  return `Open ${service.tier}`
}

function openService(service: Service) {
  router.push(`/services/${service.id}`)
}

function handleSort(key: string) {
  toggleSort(key as ServiceSortKey)
}
</script>

<template>
  <div class="service-collection-view">
    <WorkspaceHeader title="Services" subtitle="Find what is being delivered, and to whom." />

    <CollectionToolbar
      v-model:search="search"
      search-placeholder="Search by service ID, customer, address, or device serial"
    >
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseSelect v-model="technology" label="Technology" :options="technologyOptions" />
      <BaseSelect v-model="category" label="Service Type" :options="categoryOptions" />
      <BaseSelect v-model="customerType" label="Customer Type" :options="customerTypeOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="services"
        :row-key="(service) => service.id"
        :row-label="rowLabel"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No services match these filters"
        empty-description="Try a different search term or clearing a filter."
        @row-click="openService"
        @sort="handleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-service="{ row }">
          <div class="service-cell">
            <span class="service-cell__tier">{{ row.tier }}</span>
            <span class="service-cell__id">{{ row.id }}</span>
          </div>
        </template>

        <template #cell-customer="{ row }">
          <div class="customer-cell">
            <span class="customer-cell__name">{{ row.customerName }}</span>
            <span class="customer-cell__type">{{ row.customerType === 'business' ? 'Business' : 'Residential' }}</span>
          </div>
        </template>

        <template #cell-technology="{ row }">
          <BaseBadge variant="info">{{ row.technology === 'gpon' ? 'GPON' : 'XGS-PON' }}</BaseBadge>
        </template>

        <template #cell-status="{ row }">
          <BaseBadge :variant="STATUS_VARIANTS[row.status as ServiceStatus]">{{
            STATUS_LABELS[row.status as ServiceStatus]
          }}</BaseBadge>
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.service-collection-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.service-cell,
.customer-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.service-cell__tier {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.service-cell__id {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.customer-cell__name {
  color: var(--color-text-primary);
}

.customer-cell__type {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}
</style>
