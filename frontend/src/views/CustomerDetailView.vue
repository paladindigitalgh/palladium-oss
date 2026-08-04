<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { getCustomerById } from '@/services/customers/customerRepository'
import type { Customer, CustomerStatus } from '@/types/customer'

/**
 * The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
 * section 8, "Customer Workspace"). This milestone is framework
 * validation, not the Customer Workspace itself: the header binds to a
 * real customer looked up by route param (docs/09-WORKSPACE-
 * SPECIFICATIONS.md, "Canonical Detail Views" -- the same object should
 * always open the same Detail View, so proving that routing/lookup path
 * matters here), but every section body is an honest "not yet
 * implemented" placeholder rather than fabricated business data. Real
 * Summary/Services/Devices/Timeline/Notes content is the next
 * milestone's job.
 *
 * Supersedes the temporary /_demo/detail-workspace route: this is now a
 * real Detail Workspace to validate the framework against.
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
      :metadata="[`Customer #${customer.id}`, `${customer.city}, ${customer.state}`]"
    />

    <SectionCard title="Summary" icon="customers">
      <BaseEmptyState
        icon="customers"
        title="Customer summary not yet implemented"
        description="Account details, contact information, and tags will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Services" icon="services">
      <BaseEmptyState
        icon="services"
        title="Services not yet implemented"
        description="This customer's active services will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Devices" icon="devices">
      <BaseEmptyState
        icon="devices"
        title="Devices not yet implemented"
        description="Equipment assigned to this customer will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Timeline" icon="clock">
      <BaseEmptyState
        icon="clock"
        title="Timeline not yet implemented"
        description="Provisioning, workflow, and account history will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Notes">
      <BaseEmptyState
        icon="check"
        title="Notes not yet implemented"
        description="Operator notes for this customer will appear here in a future milestone."
      />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.customer-detail-view__status {
  padding: var(--space-6);
}
</style>
