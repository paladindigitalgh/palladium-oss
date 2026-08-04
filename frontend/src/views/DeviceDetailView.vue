<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import BaseIcon, { type IconName } from '@/components/base/BaseIcon.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import { getDeviceById } from '@/services/devices/deviceRepository'
import type { Device, DeviceStatus } from '@/types/device'

/**
 * The Device Detail Workspace placeholder (docs/09-WORKSPACE-
 * SPECIFICATIONS.md, section 10, "Device Workspace"). Same treatment as
 * the Customer Detail Workspace's own first pass: the header and Summary
 * bind to a real device looked up by route param (proving the Device
 * Collection -> Device Detail routing and canonical-lookup path work),
 * but Interfaces/Alarms/Timeline stay honest "not yet implemented"
 * placeholders rather than fabricated operational data. Building those
 * out is a future milestone's job.
 */
const route = useRoute()
const router = useRouter()

const device = ref<Device | null>(null)
const loading = ref(true)
const notFound = ref(false)

async function load(id: string) {
  loading.value = true
  notFound.value = false
  device.value = null

  const result = await getDeviceById(id)
  if (result) {
    device.value = result
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

const STATUS_LABELS: Record<DeviceStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  provisioning: 'Provisioning',
}

const STATUS_VARIANTS: Record<DeviceStatus, 'success' | 'error' | 'warning' | 'info'> = {
  online: 'success',
  offline: 'error',
  warning: 'warning',
  provisioning: 'info',
}

interface SummaryFact {
  icon: IconName
  label: string
  value: string
}

const summaryFacts = computed<SummaryFact[]>(() => {
  const d = device.value
  if (!d) return []
  const facts: SummaryFact[] = [
    { icon: 'devices', label: 'Type', value: d.type },
    { icon: 'health', label: 'Status', value: STATUS_LABELS[d.status] },
    { icon: 'location', label: 'Location', value: d.location },
  ]
  if (d.technology) {
    facts.push({ icon: 'network', label: 'Technology', value: d.technology === 'gpon' ? 'GPON' : 'XGS-PON' })
  }
  facts.push({
    icon: 'customers',
    label: 'Assigned Customer',
    value: d.assignedCustomerName ?? 'Unassigned',
  })
  return facts
})
</script>

<template>
  <div v-if="loading" class="device-detail-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="notFound" class="device-detail-view__status">
    <BaseErrorState
      title="Device not found"
      description="This device may have been removed, or the link may be out of date."
    >
      <BaseButton variant="secondary" @click="router.push('/devices')">Back to Devices</BaseButton>
    </BaseErrorState>
  </div>

  <DetailWorkspace v-else-if="device">
    <WorkspaceHeader
      :title="device.model"
      :subtitle="`${device.type} · ${device.location}`"
      :status="{ label: STATUS_LABELS[device.status], variant: STATUS_VARIANTS[device.status] }"
      :metadata="[`Serial ${device.serialNumber}`, device.assignedCustomerName ?? 'Unassigned']"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton
              variant="ghost"
              size="sm"
              disabled
              disabled-reason="Workflow actions are not yet implemented."
            >
              Run Diagnostics
            </BaseButton>
          </template>
          <template #primary>
            <BaseButton
              variant="primary"
              size="sm"
              disabled
              disabled-reason="Workflow actions are not yet implemented."
            >
              Reboot Device
            </BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SectionCard title="Summary" icon="devices">
      <div class="summary-facts">
        <div v-for="fact in summaryFacts" :key="fact.label" class="summary-fact">
          <BaseIcon :name="fact.icon" size="sm" class="summary-fact__icon" />
          <div class="summary-fact__text">
            <span class="summary-fact__label">{{ fact.label }}</span>
            <span class="summary-fact__value">{{ fact.value }}</span>
          </div>
        </div>
      </div>
    </SectionCard>

    <SectionCard title="Interfaces">
      <BaseEmptyState
        icon="network"
        title="Interfaces not yet implemented"
        description="Port and interface state will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Alarms" icon="alert">
      <BaseEmptyState
        icon="check"
        title="Alarms not yet implemented"
        description="Active alarms for this device will appear here in a future milestone."
      />
    </SectionCard>

    <SectionCard title="Timeline">
      <BaseEmptyState
        icon="clock"
        title="Timeline not yet implemented"
        description="Provisioning and configuration history will appear here in a future milestone."
      />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.device-detail-view__status {
  padding: var(--space-6);
}

.summary-facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-3);
}

.summary-fact {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background-color: var(--color-bg);
}

.summary-fact__icon {
  color: var(--color-text-muted);
  margin-top: 2px;
}

.summary-fact__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.summary-fact__label {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.summary-fact__value {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}
</style>
