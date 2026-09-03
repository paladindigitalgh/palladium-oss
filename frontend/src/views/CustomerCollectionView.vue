<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import CustomerFormDialog from '@/components/dialogs/CustomerFormDialog.vue'
import type { Customer } from '@/types/customer'
import type { Location } from '@/types/location'
import { useCustomerCollection, type CustomerSortKey } from '@/composables/useCustomerCollection'
import { listLocations } from '@/services/locations/locationRepository'

/**
 * The Customer Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"): discovery only. Backed by the real
 * Customer API. "Location" is resolved by fetching every Location once
 * and indexing by customerId, the same client-side-join pattern
 * locationRepository.ts documents -- there is no server-side "customer's
 * primary location" query yet.
 */
const router = useRouter()

const {
  search,
  status,
  customerType,
  sortKey,
  sortDirection,
  toggleSort,
  page,
  pageSize,
  customers,
  total,
  loading,
} = useCustomerCollection()

const locationsByCustomerId = ref<Map<string, Location>>(new Map())

onMounted(async () => {
  const locations = await listLocations()
  const byCustomer = new Map<string, Location>()
  for (const location of locations) {
    if (!byCustomer.has(location.customerId)) byCustomer.set(location.customerId, location)
  }
  locationsByCustomerId.value = byCustomer
})

const columns: DataTableColumn[] = [
  { key: 'customer', label: 'Customer', sortable: true },
  { key: 'location', label: 'Location' },
  { key: 'status', label: 'Status', sortable: true },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'Active', label: 'Active' },
  { value: 'Inactive', label: 'Inactive' },
  { value: 'Archived', label: 'Archived' },
]

const typeOptions = [
  { value: 'all', label: 'All Types' },
  { value: 'Residential', label: 'Residential' },
  { value: 'Business', label: 'Business' },
  { value: 'Government', label: 'Government' },
  { value: 'Internal', label: 'Internal' },
]

function rowLabel(customer: Customer): string {
  return `Open ${customer.name}`
}

function openCustomer(customer: Customer) {
  router.push(`/customers/${customer.id}`)
}

function handleSort(key: string) {
  toggleSort(key as CustomerSortKey)
}

const showNewCustomerDialog = ref(false)

function handleCustomerCreated(customer: Customer) {
  showNewCustomerDialog.value = false
  router.push(`/customers/${customer.id}`)
}
</script>

<template>
  <div class="customer-collection-view">
    <WorkspaceHeader title="Customers" subtitle="Search, filter, and open a customer workspace.">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showNewCustomerDialog = true">New Customer</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <CustomerFormDialog
      :open="showNewCustomerDialog"
      @close="showNewCustomerDialog = false"
      @created="handleCustomerCreated"
    />

    <CollectionToolbar v-model:search="search" search-placeholder="Search by name or customer ID">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
      <BaseSelect v-model="customerType" label="Customer Type" :options="typeOptions" />
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
            <span class="customer-cell__meta">{{ row.id }} · {{ row.customerType }}</span>
          </div>
        </template>

        <template #cell-location="{ row }">
          <span v-if="locationsByCustomerId.get(row.id)" class="location-cell">
            {{ locationsByCustomerId.get(row.id)!.city }}, {{ locationsByCustomerId.get(row.id)!.state }}
          </span>
          <span v-else class="location-cell location-cell--empty">No location on file</span>
        </template>

        <template #cell-status="{ row }">
          {{ row.status }}
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

.customer-cell {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.customer-cell__name {
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.customer-cell__meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.location-cell {
  color: var(--color-text-primary);
}

.location-cell--empty {
  color: var(--color-text-muted);
}
</style>
