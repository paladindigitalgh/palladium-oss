<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import type { Customer } from '@/types/customer'
import { useCustomerCollection, type CustomerSortKey } from '@/composables/useCustomerCollection'

/**
 * The Customer Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"): discovery only. It exists to help an
 * operator find a customer and open that customer's Detail Workspace --
 * not to summarize account health or duplicate what the Detail Workspace
 * will show. Three columns, per that document's own example ("Customers:
 * Customer, Location, Primary Service"), and nothing else.
 *
 * This is a plain page, not a DetailWorkspace: a Collection View has no
 * single selected object and no sections
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Workspace Architecture": "An
 * Entity Workspace's Collection View is a different, simpler shape --
 * typically a DataTable plus search/filter/sort controls").
 */
const router = useRouter()

const {
  search,
  status,
  serviceTechnology,
  customerType,
  city,
  cities,
  sortKey,
  sortDirection,
  toggleSort,
  page,
  pageSize,
  customers,
  total,
  loading,
} = useCustomerCollection()

const columns: DataTableColumn[] = [
  { key: 'customer', label: 'Customer', sortable: true },
  { key: 'location', label: 'Location', sortable: true },
  { key: 'primaryService', label: 'Primary Service', sortable: true },
]

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'pending', label: 'Pending' },
  { value: 'cancelled', label: 'Cancelled' },
  { value: 'all', label: 'All Statuses' },
]

const serviceOptions = [
  { value: 'any', label: 'Any' },
  { value: 'gpon', label: 'GPON' },
  { value: 'xgs-pon', label: 'XGS-PON' },
]

const typeOptions = [
  { value: 'all', label: 'All Types' },
  { value: 'residential', label: 'Residential' },
  { value: 'business', label: 'Business' },
]

const cityOptions = computed(() => [
  { value: 'all', label: 'All Locations' },
  ...cities.map((option) => ({ value: option, label: option })),
])

function customerTypeLabel(customer: Customer): string {
  return customer.type === 'business' ? 'Business' : 'Residential'
}

function rowLabel(customer: Customer): string {
  return `Open ${customer.name}`
}

function openCustomer(customer: Customer) {
  router.push(`/customers/${customer.id}`)
}

function handleSort(key: string) {
  toggleSort(key as CustomerSortKey)
}
</script>

<template>
  <div class="customer-collection-view">
    <WorkspaceHeader title="Customers" subtitle="Search, filter, and open a customer workspace." />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name, customer ID, or address">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseSelect v-model="serviceTechnology" label="Service Type" :options="serviceOptions" />
      <BaseSelect v-model="customerType" label="Customer Type" :options="typeOptions" />
      <BaseSelect v-model="city" label="Location" :options="cityOptions" />
    </CollectionToolbar>

    <BaseCard :padded="false">
      <DataTable
        :columns="columns"
        :rows="customers"
        :row-key="(customer) => customer.id"
        :row-label="rowLabel"
        :loading="loading"
        :sort-key="sortKey"
        :sort-direction="sortDirection"
        :page="page"
        :page-size="pageSize"
        :total="total"
        empty-title="No customers match these filters"
        empty-description="Try a different search term or clearing a filter."
        @row-click="openCustomer"
        @sort="handleSort"
        @update:page="(next) => (page = next)"
      >
        <template #cell-customer="{ row }">
          <div class="customer-cell">
            <span class="customer-cell__name">{{ row.name }}</span>
            <span class="customer-cell__meta">
              {{ row.id }}
              <BaseBadge variant="neutral">{{ customerTypeLabel(row) }}</BaseBadge>
            </span>
          </div>
        </template>

        <template #cell-location="{ row }">
          <div class="location-cell">
            <span class="location-cell__city">{{ row.city }}, {{ row.state }}</span>
            <span class="location-cell__address">{{ row.address }}</span>
          </div>
        </template>

        <template #cell-primaryService="{ row }">
          <div class="service-cell">
            <span class="service-cell__tier">{{ row.primaryService.tier }}</span>
            <BaseBadge variant="info">{{ row.primaryService.technology === 'gpon' ? 'GPON' : 'XGS-PON' }}</BaseBadge>
          </div>
        </template>
      </DataTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.customer-collection-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.customer-cell,
.location-cell,
.service-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.customer-cell__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.customer-cell__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.location-cell__city {
  color: var(--color-text-primary);
}

.location-cell__address {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.service-cell {
  flex-direction: row;
  align-items: center;
  gap: var(--space-2);
}

.service-cell__tier {
  color: var(--color-text-primary);
}
</style>
