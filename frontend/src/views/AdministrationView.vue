<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import PlanFormDialog from '@/components/dialogs/PlanFormDialog.vue'
import { listProducts } from '@/services/products/productRepository'
import { listProvisioningProfiles } from '@/services/provisioningProfiles/provisioningProfileRepository'
import type { Product } from '@/types/product'
import type { ProvisioningProfile } from '@/types/provisioningProfile'

/**
 * The Administration Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md
 * section 16): "Primary Panels," not a Detail Workspace -- see
 * PlaceholderWorkspaceView.vue's own doc comment on why every route here
 * uses a plain header instead of DetailWorkspace. Plans is the first real
 * panel; System Health, User Management, Plugin Management, and the rest
 * of that section's list remain unbuilt (still whatever
 * PlaceholderWorkspaceView.vue would have shown), not implied by this
 * file.
 *
 * A "Plan" is a Product (docs/03-DOMAIN-MODEL.md section 5 -- what the
 * ISP sells, e.g. "Residential Internet 500/500") paired with a
 * ProvisioningProfile (internal/provisioning -- which OLT vendor profile
 * an operator already built by hand delivers it). Both already have full
 * backend CRUD; this panel is their first real management UI on either
 * domain (see productRepository.ts's and PlanFormDialog.vue's own doc
 * comments). GET /products and GET /provisioning-profiles each have no
 * server-side filtering, so both lists are fetched in full and joined
 * client-side by productId -- fine at this domain's expected size (one
 * row per Product per vendor).
 */
const products = ref<Product[]>([])
const profilesByProductId = ref<Map<string, ProvisioningProfile[]>>(new Map())
const loading = ref(true)

async function load() {
  loading.value = true
  const [productList, profileList] = await Promise.all([listProducts(), listProvisioningProfiles()])
  products.value = productList

  const byProductId = new Map<string, ProvisioningProfile[]>()
  for (const profile of profileList) {
    const existing = byProductId.get(profile.productId) ?? []
    existing.push(profile)
    byProductId.set(profile.productId, existing)
  }
  profilesByProductId.value = byProductId

  loading.value = false
}

onMounted(load)

const planColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Name' },
  { key: 'category', label: 'Category' },
  { key: 'profiles', label: 'OLT Profiles' },
  { key: 'status', label: 'Status' },
]

const profilesForProduct = computed(() => (productId: string) => profilesByProductId.value.get(productId) ?? [])

const showPlanForm = ref(false)

function handlePlanCreated(payload: { product: Product; profile: ProvisioningProfile }) {
  showPlanForm.value = false
  products.value = [...products.value, payload.product]
  const existing = profilesByProductId.value.get(payload.product.id) ?? []
  profilesByProductId.value = new Map(profilesByProductId.value).set(payload.product.id, [...existing, payload.profile])
}
</script>

<template>
  <div class="administration-view">
    <WorkspaceHeader title="Administration" subtitle="Platform configuration and administrative tasks." />

    <BaseCard>
      <div class="panel-header">
        <h2 class="panel-title">Plans</h2>
        <BaseButton variant="secondary" size="sm" @click="showPlanForm = true">New Plan</BaseButton>
      </div>
      <p class="panel-description">
        A Plan is a commercial offering (a Product) paired with the OLT vendor profile that delivers it. Build the
        profile on the OLT first, then create the matching Plan here.
      </p>

      <PlanFormDialog :open="showPlanForm" @close="showPlanForm = false" @created="handlePlanCreated" />

      <div v-if="loading" class="panel-status">
        <BaseLoadingState :lines="4" />
      </div>

      <SimpleTable
        v-else
        :columns="planColumns"
        :rows="products"
        :row-key="(product) => product.id"
        empty-icon="settings"
        empty-title="No plans yet"
      >
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-category="{ row }">{{ row.category }}</template>
        <template #cell-profiles="{ row }">
          <span v-if="profilesForProduct(row.id).length === 0" class="no-profile">No OLT profile mapped</span>
          <span v-else class="cell-mono">
            {{ profilesForProduct(row.id).map((profile) => `${profile.vendor}: ${profile.profileName}`).join(', ') }}
          </span>
        </template>
        <template #cell-status="{ row }">{{ row.status }}</template>
      </SimpleTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.administration-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}

.panel-title {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.panel-description {
  margin: 0 0 var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.panel-status {
  padding: var(--space-4) 0;
}

.cell-mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.no-profile {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
