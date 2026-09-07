<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import BaseDisclosure from '@/components/base/BaseDisclosure.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import PlanFormDialog from '@/components/dialogs/PlanFormDialog.vue'
import ProviderFormDialog from '@/components/dialogs/ProviderFormDialog.vue'
import { listProducts } from '@/services/products/productRepository'
import { listProvisioningProfiles } from '@/services/provisioningProfiles/provisioningProfileRepository'
import { listProviders } from '@/services/providers/providerRepository'
import type { Product } from '@/types/product'
import type { ProvisioningProfile } from '@/types/provisioningProfile'
import type { Provider } from '@/types/provider'

/**
 * The Providers page, reached from the Administration landing hub
 * (AdministrationView.vue). Each Provider (internal/provider -- the
 * retail ISP identity a Plan belongs to) is a top-level BaseDisclosure;
 * expanding one reveals that Provider's Plans nested inside, per the
 * user's explicit request. Split out of what was previously a combined
 * Providers+Plans stacked-panel page into its own route
 * (/administration/providers); see AdministrationView.vue's own doc
 * comment for why this deviates from
 * docs/09-WORKSPACE-SPECIFICATIONS.md section 16's original description
 * of Administration.
 *
 * A "Plan" is a Product (docs/03-DOMAIN-MODEL.md section 21) paired with
 * a ProvisioningProfile (internal/provisioning). GET /products,
 * GET /provisioning-profiles, and GET /providers each have no
 * server-side filtering, so all three lists are fetched in full and
 * joined client-side -- fine at this domain's expected size (one row
 * per Product per vendor, one row per ISP).
 *
 * "New Plan" opens PlanFormDialog.vue scoped to the Provider whose
 * section it was opened from by passing a single-element `providers`
 * array: that dialog's own reset() already defaults providerId to
 * `providers[0]`, and its Provider picker only renders when
 * `providers.length > 1` -- passing one Provider hides the picker and
 * locks the choice, with no changes needed to that dialog.
 */
const products = ref<Product[]>([])
const profilesByProductId = ref<Map<string, ProvisioningProfile[]>>(new Map())
const providers = ref<Provider[]>([])
const loading = ref(true)

async function load() {
  loading.value = true
  const [productList, profileList, providerList] = await Promise.all([
    listProducts(),
    listProvisioningProfiles(),
    listProviders(),
  ])
  products.value = productList
  providers.value = providerList

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
const plansForProvider = computed(() => (providerId: string) => products.value.filter((p) => p.providerId === providerId))

// --- Providers ---

const showProviderForm = ref(false)

function handleProviderCreated(provider: Provider) {
  showProviderForm.value = false
  providers.value = [...providers.value, provider]
}

// --- Plans ---

// Which Provider's "New Plan" dialog is open, if any -- a single shared
// dialog instance scoped to whichever Provider's section it was opened
// from, rather than one PlanFormDialog per Provider.
const planFormProvider = ref<Provider | null>(null)

function handlePlanCreated(payload: { product: Product; profile: ProvisioningProfile }) {
  planFormProvider.value = null
  products.value = [...products.value, payload.product]
  const existing = profilesByProductId.value.get(payload.product.id) ?? []
  profilesByProductId.value = new Map(profilesByProductId.value).set(payload.product.id, [...existing, payload.profile])
}
</script>

<template>
  <div class="administration-providers-view">
    <WorkspaceHeader title="Providers" :breadcrumbs="[{ label: 'Administration', to: '/administration' }]">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showProviderForm = true">New Provider</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <p class="page-description">
      A Provider is a retail ISP identity Plans belong to. Only relevant once more than one ISP shares this network
      (open-access) -- with a single Provider, this page is a one-time setup step. Expand a Provider to see and
      manage its Plans.
    </p>

    <ProviderFormDialog :open="showProviderForm" @close="showProviderForm = false" @created="handleProviderCreated" />

    <PlanFormDialog
      :open="planFormProvider !== null"
      :providers="planFormProvider ? [planFormProvider] : []"
      @close="planFormProvider = null"
      @created="handlePlanCreated"
    />

    <div v-if="loading" class="page-status">
      <BaseLoadingState :lines="4" />
    </div>

    <template v-else>
      <p v-if="providers.length === 0" class="no-providers">No providers yet.</p>

      <BaseDisclosure v-for="provider in providers" :key="provider.id" :title="provider.name">
        <template #extra>
          <BaseBadge :variant="provider.status === 'Active' ? 'success' : 'neutral'">{{ provider.status }}</BaseBadge>
        </template>

        <div class="provider-plans">
          <div class="provider-plans__header">
            <BaseButton variant="secondary" size="sm" @click="planFormProvider = provider">New Plan</BaseButton>
          </div>

          <SimpleTable
            :columns="planColumns"
            :rows="plansForProvider(provider.id)"
            :row-key="(product) => product.id"
            empty-icon="settings"
            empty-title="No plans yet for this provider"
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
        </div>
      </BaseDisclosure>
    </template>
  </div>
</template>

<style scoped>
.administration-providers-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-description {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.page-status {
  padding: var(--space-4) 0;
}

.no-providers {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.provider-plans {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.provider-plans__header {
  display: flex;
  justify-content: flex-end;
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
