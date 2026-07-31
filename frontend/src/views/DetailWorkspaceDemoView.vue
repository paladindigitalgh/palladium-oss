<script setup lang="ts">
import DetailWorkspace from '@/components/workspace/DetailWorkspace.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import SectionCard from '@/components/data-display/SectionCard.vue'
import BasePropertyGrid from '@/components/base/BasePropertyGrid.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import TimelineEntries from '@/components/data-display/TimelineEntries.vue'

/**
 * TEMPORARY. This route exists only to validate the Detail Workspace
 * framework (layout, Contents navigation, scroll behavior, section
 * collapse, active-section tracking, responsive behavior) in isolation,
 * with placeholder content -- it is not linked from primary navigation
 * (see router/index.ts) and should be deleted once a real Detail
 * Workspace (starting with Customers) exists to demonstrate the
 * framework instead.
 *
 * The "John Smith / Residential Customer" placeholder values are lifted
 * directly from this milestone's own example of the desired developer
 * experience -- not an implementation of Customer business logic. There
 * is no data fetching, no customer model, nothing beyond static strings.
 */
const PLACEHOLDER_SUMMARY = [
  { label: 'Account Number', value: '000000-DEMO' },
  { label: 'Plan', value: 'Example Residential Plan' },
  { label: 'Address', value: '123 Example Street' },
  { label: 'Primary Contact', value: 'demo@example.test' },
]

const PLACEHOLDER_TIMELINE = [
  { id: 't1', label: 'Example event created', timestamp: '2 minutes ago' },
  { id: 't2', label: 'Example status changed', timestamp: '1 hour ago' },
  { id: 't3', label: 'Workspace framework validated', timestamp: 'Yesterday' },
]
</script>

<template>
  <DetailWorkspace>
    <WorkspaceHeader
      title="John Smith"
      subtitle="Residential Customer"
      :status="{ label: 'Active', variant: 'success' }"
      :metadata="['Account #000000-DEMO', 'Member since 2023', 'West Region']"
    >
      <template #actions>
        <WorkspaceActions>
          <template #secondary>
            <BaseButton variant="ghost" size="sm">Example secondary action</BaseButton>
          </template>
          <template #primary>
            <BaseButton variant="primary" size="sm">Example primary action</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <SectionCard title="Summary" icon="customers">
      <BasePropertyGrid :properties="PLACEHOLDER_SUMMARY" />
    </SectionCard>

    <SectionCard title="Services" icon="services" badge="2">
      <ul class="demo-list">
        <li class="demo-list__item">Example Residential Internet -- 500 Mbps</li>
        <li class="demo-list__item">Example Static IP Add-on</li>
      </ul>
    </SectionCard>

    <SectionCard title="Devices" icon="devices" badge="1">
      <ul class="demo-list">
        <li class="demo-list__item">Example ONU -- Serial DEMO-0001</li>
      </ul>
    </SectionCard>

    <SectionCard title="Timeline" icon="clock">
      <TimelineEntries :entries="PLACEHOLDER_TIMELINE" />
    </SectionCard>

    <SectionCard title="Notes">
      <BaseEmptyState
        icon="check"
        title="No notes yet"
        description="Notes added to this workspace will appear here."
      />
    </SectionCard>
  </DetailWorkspace>
</template>

<style scoped>
.demo-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.demo-list__item {
  padding: var(--space-2) 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border);
}

.demo-list__item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}
</style>
