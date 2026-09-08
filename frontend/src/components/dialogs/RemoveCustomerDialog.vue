<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import { getCustomerRemovalPreview, executeCustomerRemoval } from '@/services/customers/customerRemovalRepository'
import { ApiError } from '@/services/api/httpClient'
import type { CustomerRemovalPreview } from '@/types/customerRemoval'

/**
 * "Remove Customer" (internal/customer/removal's own doc comment):
 * previews the full cascade -- every Location, Service, and piece of
 * equipment affected -- before the operator confirms, then executes it.
 * Nothing shown here or done by Execute is a delete: Locations go
 * Inactive, Services go Disconnected, equipment is unassigned. The
 * underlying Device is never touched, and any equipment whose row says
 * it will run an OLT teardown means a real SSH command against real
 * hardware, not just a database update -- that is called out explicitly
 * rather than folded into a generic "N services removed" count.
 */
const props = defineProps<{ open: boolean; customerId: string; customerName: string }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'removed'): void
}>()

const loading = ref(false)
const loadError = ref<string | null>(null)
const preview = ref<CustomerRemovalPreview | null>(null)

const pending = ref(false)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  loadError.value = null
  preview.value = null
  try {
    preview.value = await getCustomerRemovalPreview(props.customerId)
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : 'The removal preview could not be loaded.'
  } finally {
    loading.value = false
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) load()
  },
)

const serviceCount = () => preview.value?.locations.reduce((sum, loc) => sum + loc.services.length, 0) ?? 0
const oltTeardownCount = () =>
  preview.value?.locations.reduce(
    (sum, loc) => sum + loc.services.reduce((s, svc) => s + svc.equipment.filter((eq) => eq.willRunOltTeardown).length, 0),
    0,
  ) ?? 0

async function confirmRemove() {
  pending.value = true
  error.value = null
  try {
    await executeCustomerRemoval(props.customerId)
    emit('removed')
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'The customer could not be removed.'
  } finally {
    pending.value = false
  }
}

function close() {
  emit('close')
}
</script>

<template>
  <BaseModal :open="open" title="Remove Customer" @close="close">
    <div class="remove-customer">
      <p class="remove-customer__intro">
        Removing {{ customerName }} deactivates every Location and disconnects every Service below. Equipment is
        unassigned, not deleted — devices stay in Inventory.
      </p>

      <div v-if="loading" class="remove-customer__status">
        <BaseLoadingState :lines="4" />
      </div>

      <p v-else-if="loadError" class="remove-customer__error" role="alert">{{ loadError }}</p>

      <template v-else-if="preview">
        <BaseEmptyState
          v-if="preview.locations.length === 0"
          icon="customers"
          title="Nothing attached"
          description="This customer has no locations or services to remove."
        />

        <template v-else>
          <p v-if="oltTeardownCount() > 0" class="remove-customer__warning" role="alert">
            {{ oltTeardownCount() }} piece(s) of equipment currently have a service profile applied on their OLT — removing
            this customer will run a real command to take that profile off, not just update the database.
          </p>

          <ul class="remove-customer__locations">
            <li v-for="loc in preview.locations" :key="loc.locationId" class="remove-customer__location">
              <div class="remove-customer__location-header">
                <span class="remove-customer__name">{{ loc.name }}</span>
                <span class="remove-customer__meta">{{ loc.status }} → Inactive</span>
              </div>

              <ul v-if="loc.services.length" class="remove-customer__services">
                <li v-for="svc in loc.services" :key="svc.serviceId" class="remove-customer__service">
                  <div class="remove-customer__service-header">
                    <span>{{ svc.description || `Service ${svc.serviceId}` }}</span>
                    <span class="remove-customer__meta">{{ svc.status }} → Disconnected</span>
                  </div>
                  <ul v-if="svc.equipment.length" class="remove-customer__equipment">
                    <li v-for="eq in svc.equipment" :key="eq.serviceEquipmentId">
                      {{ eq.deviceName }} ({{ eq.role }})
                      <span v-if="eq.willRunOltTeardown" class="remove-customer__teardown-tag">removes OLT profile</span>
                    </li>
                  </ul>
                </li>
              </ul>
              <p v-else class="remove-customer__meta">No services at this location.</p>
            </li>
          </ul>
        </template>
      </template>

      <p v-if="error" class="remove-customer__error" role="alert">{{ error }}</p>

      <div class="remove-customer__actions">
        <BaseButton type="button" variant="secondary" :disabled="pending" @click="close">Cancel</BaseButton>
        <BaseButton
          variant="destructive"
          :disabled="pending || loading || !!loadError"
          @click="confirmRemove"
        >
          {{ pending ? 'Removing…' : `Remove Customer (${serviceCount()} service${serviceCount() === 1 ? '' : 's'})` }}
        </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<style scoped>
.remove-customer {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.remove-customer__intro {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.remove-customer__status {
  padding: var(--space-2) 0;
}

.remove-customer__warning {
  font-size: var(--font-size-sm);
  color: var(--color-warning);
}

.remove-customer__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.remove-customer__locations {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-height: 320px;
  overflow-y: auto;
}

.remove-customer__location {
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.remove-customer__location-header,
.remove-customer__service-header {
  display: flex;
  justify-content: space-between;
  gap: var(--space-2);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.remove-customer__services {
  margin-top: var(--space-2);
  padding-left: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.remove-customer__service-header {
  font-weight: var(--font-weight-medium);
}

.remove-customer__equipment {
  margin-top: 2px;
  padding-left: var(--space-4);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.remove-customer__meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  font-weight: var(--font-weight-normal);
}

.remove-customer__teardown-tag {
  margin-left: var(--space-2);
  color: var(--color-warning);
}

.remove-customer__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-2);
}
</style>
