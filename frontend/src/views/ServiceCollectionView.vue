<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import CollectionToolbar from '@/components/data-display/CollectionToolbar.vue'
import DataTable, { type DataTableColumn } from '@/components/data-display/DataTable.vue'
import type { Service } from '@/types/service'
import type { Customer } from '@/types/customer'
import { useServiceCollection, type ServiceSortKey } from '@/composables/useServiceCollection'
import { listLocations } from '@/services/locations/locationRepository'
import { listCustomers } from '@/services/customers/customerRepository'
import { resolveServiceLabels } from '@/services/services/serviceLabels'

/**
 * The Service Collection View (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * "Collection View & Detail View"). Backed by the real Service API. The
 * owning Customer is resolved by joining Service -> Location -> Customer
 * client-side (fetched once), since a Service carries only locationId,
 * never a customer reference directly (docs/03-DOMAIN-MODEL.md: a
 * Customer owns Services through Locations).
 */
const router = useRouter()

const { search, status, sortKey, sortDirection, toggleSort, page, pageSize, services, total, loading } = useServiceCollection()

const customerByLocationId = ref<Map<string, Customer>>(new Map())
const serviceLabelsById = ref<Map<string, string>>(new Map())

onMounted(async () => {
  const [locations, { items: customers }] = await Promise.all([listLocations(), listCustomers({ pageSize: 1000 })])
  const customersById = new Map(customers.map((customer) => [customer.id, customer]))
  const byLocation = new Map<string, Customer>()
  for (const location of locations) {
    const customer = customersById.get(location.customerId)
    if (customer) byLocation.set(location.id, customer)
  }
  customerByLocationId.value = byLocation
})

// Re-resolved whenever the visible page of Services changes (search,
// filter, sort, or page navigation all reassign `services` inside
// useServiceCollection) -- cheap regardless, since resolveServiceLabels
// is one pair of requests no matter how many Services are passed in.
watch(
  services,
  async (currentServices) => {
    serviceLabelsById.value = await resolveServiceLabels(currentServices)
  },
  { immediate: true },
)

const columns: DataTableColumn[] = [
  { key: 'service', label: 'Service', sortable: true },
  { key: 'customer', label: 'Customer' },
  { key: 'status', label: 'Status', sortable: true },
]

const statusOptions = [
  { value: 'all', label: 'All Statuses' },
  { value: 'Pending', label: 'Pending' },
  { value: 'Active', label: 'Active' },
  { value: 'Suspended', label: 'Suspended' },
  { value: 'Disconnected', label: 'Disconnected' },
]

function rowLabel(service: Service): string {
  return `Open ${serviceLabelsById.value.get(service.id) ?? service.id}`
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

    <CollectionToolbar v-model:search="search" search-placeholder="Search by service ID or description">
      <BaseSelect v-model="status" label="Status" :options="statusOptions" />
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
          <span class="service-cell__id">{{ serviceLabelsById.get(row.id) ?? row.id }}</span>
        </template>

        <template #cell-customer="{ row }">
          <span v-if="customerByLocationId.get(row.locationId)" class="customer-cell">
            {{ customerByLocationId.get(row.locationId)!.name }}
          </span>
          <span v-else class="customer-cell customer-cell--empty">Unknown</span>
        </template>

        <template #cell-status="{ row }">
          {{ row.status }}
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

.service-cell__id {
  color: var(--color-text-primary);
}

.customer-cell {
  color: var(--color-text-primary);
}

.customer-cell--empty {
  color: var(--color-text-muted);
}
</style>
