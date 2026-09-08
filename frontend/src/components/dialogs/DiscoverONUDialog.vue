<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseModal from '@/components/base/BaseModal.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import { getAggregatedBlacklist } from '@/services/diagnostics/diagnosticsRepository'
import { authorizeONU } from '@/services/provisioning/provisioningRepository'
import { getDeviceBySerialNumber } from '@/services/devices/deviceRepository'
import { ApiError } from '@/services/api/httpClient'
import type { BlacklistedONU, UnreachableOLT } from '@/types/onuDiagnostics'

/**
 * The "Discover ONU" picker (TASKS.md Phase 4: turns
 * internal/diagnostics/kontron/service.KontronService.AggregatedBlacklist
 * and AuthorizeONU -- both real since earlier this session, neither ever
 * wired to a UI -- into something an operator can actually use).
 *
 * Authorizing a row always runs against the OLT: appearing in the
 * blacklist means the OLT currently has this serial number unauthorized
 * right now, regardless of what Palladium's own Device table says. What
 * happens *after* authorization branches on whether a Device record for
 * this serial number already exists (see getDeviceBySerialNumber's own
 * doc comment on why that check matters): if one does, this dialog links
 * to it directly instead of creating a duplicate; if not, it hands off
 * to the caller's own DeviceFormDialog via the `authorized` event,
 * pre-filled with the serial number, rather than duplicating that form's
 * fields here.
 */
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  (event: 'close'): void
  (event: 'authorized', payload: { serialNumber: string }): void
}>()

const router = useRouter()

const loading = ref(false)
const loadError = ref<string | null>(null)
const onus = ref<BlacklistedONU[]>([])
const unreachableOlts = ref<UnreachableOLT[]>([])

const authorizingSerial = ref<string | null>(null)
const rowError = ref<string | null>(null)
const existingDevice = ref<{ serialNumber: string; deviceId: string; interface: string } | null>(null)

async function load() {
  loading.value = true
  loadError.value = null
  onus.value = []
  unreachableOlts.value = []
  existingDevice.value = null
  try {
    const result = await getAggregatedBlacklist()
    onus.value = result.onus
    unreachableOlts.value = result.unreachableOlts
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : 'The blacklist scan could not be completed.'
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

async function authorize(onu: BlacklistedONU) {
  authorizingSerial.value = onu.serialNumber
  rowError.value = null
  existingDevice.value = null
  try {
    const iface = await authorizeONU(onu.oltId, onu.interface, onu.serialNumber)
    onus.value = onus.value.filter((candidate) => candidate.serialNumber !== onu.serialNumber)

    const existing = await getDeviceBySerialNumber(onu.serialNumber)
    if (existing) {
      existingDevice.value = { serialNumber: onu.serialNumber, deviceId: existing.id, interface: iface }
    } else {
      emit('authorized', { serialNumber: onu.serialNumber })
    }
  } catch (err) {
    rowError.value = err instanceof ApiError ? err.message : 'This ONU could not be authorized.'
  } finally {
    authorizingSerial.value = null
  }
}

function viewExistingDevice() {
  if (!existingDevice.value) return
  router.push(`/devices/${existingDevice.value.deviceId}`)
  emit('close')
}

function close() {
  emit('close')
}
</script>

<template>
  <BaseModal :open="open" title="Discover ONU" @close="close">
    <div class="discover-onu">
      <p class="discover-onu__intro">
        Physically-detected ONUs that are not yet authorized on any Kontron OLT. Authorizing one clears it from this
        list and brings it into service.
      </p>

      <div v-if="loading" class="discover-onu__status">
        <BaseLoadingState :lines="3" />
      </div>

      <p v-else-if="loadError" class="discover-onu__error" role="alert">{{ loadError }}</p>

      <template v-else>
        <p v-if="unreachableOlts.length" class="discover-onu__warning" role="alert">
          Could not reach: {{ unreachableOlts.map((u) => `${u.oltName} (${u.reason})`).join(', ') }}
        </p>

        <div v-if="existingDevice" class="discover-onu__existing">
          <p>
            Authorized {{ existingDevice.serialNumber }} on {{ existingDevice.interface }}. This serial number already
            has a Device record.
          </p>
          <BaseButton variant="secondary" size="sm" @click="viewExistingDevice">View Device</BaseButton>
        </div>

        <BaseEmptyState
          v-if="onus.length === 0 && !existingDevice"
          icon="devices"
          title="Nothing to discover"
          description="No physically-detected, unauthorized ONUs were found."
        />

        <ul v-else class="discover-onu__list">
          <li v-for="onu in onus" :key="onu.serialNumber" class="discover-onu__row">
            <div class="discover-onu__row-info">
              <span class="discover-onu__serial">{{ onu.serialNumber }}</span>
              <span class="discover-onu__meta">{{ onu.oltName }} &middot; {{ onu.interface }}</span>
              <span class="discover-onu__cause">{{ onu.cause }}</span>
            </div>
            <BaseButton
              variant="primary"
              size="sm"
              :disabled="authorizingSerial !== null"
              @click="authorize(onu)"
            >
              {{ authorizingSerial === onu.serialNumber ? 'Authorizing…' : 'Authorize' }}
            </BaseButton>
          </li>
        </ul>

        <p v-if="rowError" class="discover-onu__error" role="alert">{{ rowError }}</p>
      </template>

      <div class="discover-onu__actions">
        <BaseButton type="button" variant="secondary" @click="close">Close</BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<style scoped>
.discover-onu {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.discover-onu__intro {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.discover-onu__status {
  padding: var(--space-2) 0;
}

.discover-onu__warning {
  font-size: var(--font-size-sm);
  color: var(--color-warning);
}

.discover-onu__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.discover-onu__existing {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
}

.discover-onu__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 320px;
  overflow-y: auto;
}

.discover-onu__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.discover-onu__row-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.discover-onu__serial {
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.discover-onu__meta,
.discover-onu__cause {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.discover-onu__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
