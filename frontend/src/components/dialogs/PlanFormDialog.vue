<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseInput from '@/components/base/BaseInput.vue'
import BaseSelect from '@/components/base/BaseSelect.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { listCatalogs } from '@/services/catalogs/catalogRepository'
import { createProduct } from '@/services/products/productRepository'
import { createProvisioningProfile } from '@/services/provisioningProfiles/provisioningProfileRepository'
import { ApiError } from '@/services/api/httpClient'
import type { Product, ProductCategory } from '@/types/product'
import type { ProvisioningProfile } from '@/types/provisioningProfile'

/**
 * Creates a "Plan": a Product (what the ISP sells, e.g. "Residential
 * Internet 500/500") paired with the ProvisioningProfile mapping it to
 * the OLT profile an operator already built by hand (see
 * internal/provisioning's package doc comment) -- one guided step in the
 * UI, two records underneath, so Product and ProvisioningProfile stay
 * separate domains (a Product is never vendor-specific;
 * see internal/product's own doc comment) while the operator experiences
 * "create a plan" as a single action, exactly as described when this
 * panel was designed.
 *
 * Create-only, like AssignServiceEquipmentDialog.vue: there is no
 * "edit a plan" flow yet, only "add a new one." If creating the
 * ProvisioningProfile fails after the Product has already been created
 * (e.g. the profile name is already claimed by another Product -- see
 * the migration's UNIQUE (vendor, profile_name)), the Product is left in
 * place rather than rolled back: it is still a valid, retirable Product,
 * and the operator can add a corrected profile mapping to it separately
 * without having to redo the whole form.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'created', payload: { product: Product; profile: ProvisioningProfile }): void
}>()

const name = ref('')
const category = ref<ProductCategory>('Internet')
const description = ref('')
const vendor = ref('Kontron')
const profileName = ref('')
const defaultCatalogId = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const categoryOptions: { value: ProductCategory; label: string }[] = [
  { value: 'Internet', label: 'Internet' },
  { value: 'Voice', label: 'Voice' },
  { value: 'IPTV', label: 'IPTV' },
  { value: 'Transport', label: 'Transport' },
  { value: 'ManagedWiFi', label: 'Managed WiFi' },
  { value: 'Other', label: 'Other' },
]

function reset() {
  name.value = ''
  category.value = 'Internet'
  description.value = ''
  vendor.value = 'Kontron'
  profileName.value = ''
  error.value = null
}

// The Catalog a new Plan's Product is filed under is not something this
// form asks about -- there is exactly one in practice today, and this
// panel is about speed tiers and OLT profiles, not catalog management
// (see AdministrationView.vue). The first Catalog found becomes the
// silent default, the same "fetch on open, default to first result"
// pattern AssignServiceEquipmentDialog.vue's Device picker uses.
watch(
  () => props.open,
  async (open) => {
    if (!open) return
    reset()
    const catalogs = await listCatalogs()
    defaultCatalogId.value = catalogs[0]?.id ?? ''
  },
  { immediate: true },
)

function close() {
  emit('close')
}

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
    const product = await createProduct({
      catalogId: defaultCatalogId.value,
      name: name.value,
      category: category.value,
      description: description.value,
    })
    const profile = await createProvisioningProfile({
      productId: product.id,
      vendor: vendor.value,
      profileName: profileName.value,
      description: '',
    })
    reset()
    emit('created', { product, profile })
  } catch (err) {
    error.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'That OLT profile name is already mapped to another plan.'
        : 'The plan could not be created.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <BaseModal :open="open" title="New Plan" @close="close">
    <form class="plan-form" @submit.prevent="handleSubmit">
      <BaseInput v-model="name" label="Name" placeholder="Residential Internet 500/500" required />
      <BaseSelect v-model="category" label="Category" :options="categoryOptions" />
      <BaseInput v-model="description" label="Description" />
      <BaseInput v-model="vendor" label="OLT Vendor" required />
      <BaseInput v-model="profileName" label="OLT Profile Name" placeholder="RES-500M" required />

      <p v-if="error" class="plan-form__error" role="alert">{{ error }}</p>

      <div class="plan-form__actions">
        <BaseButton type="button" variant="secondary" :disabled="submitting" @click="close">Cancel</BaseButton>
        <BaseButton type="submit" variant="primary" :disabled="submitting || !name || !profileName">
          {{ submitting ? 'Creating…' : 'Create Plan' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<style scoped>
.plan-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.plan-form__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.plan-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
