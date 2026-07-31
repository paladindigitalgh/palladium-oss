<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import StatisticCard from '@/components/data-display/StatisticCard.vue'
import DashboardWidget from '@/components/dashboard/DashboardWidget.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BasePropertyGrid from '@/components/base/BasePropertyGrid.vue'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * Milestone 2: the first real workspace. Everything below this line is
 * placeholder data, not a live query -- see the PLACEHOLDER_* constants.
 *
 * This view does not use DetailWorkspace
 * (components/workspace/DetailWorkspace.vue). This is not a deviation:
 * docs/11-COMPONENT-ARCHITECTURE.md ("Workspace Archetypes") and
 * docs/09-WORKSPACE-SPECIFICATIONS.md (section 4, same name) document
 * Dashboard as Palladium's one Landing Workspace, and DetailWorkspace is
 * the standard for Entity Workspaces (Customers, Services, Devices,
 * Network, Inventory, Explorer, Administration, and their detail-level
 * workspaces like Customer, Service, Device, OLT, Site), which manage a
 * selected object broken into collapsible sections. Dashboard has no
 * selected object and no sections: its layout is a four-card summary row
 * plus a full-width 2x2 widget grid, so there is nothing for
 * DetailWorkspace's Contents navigation to list, and no object for a
 * SectionCard to describe.
 *
 * WorkspaceHeader is still reused for the title/subtitle region, so the
 * page-heading treatment stays consistent with every other workspace.
 */
const route = useRoute()
const subtitle = computed(() =>
  typeof route.meta.description === 'string' ? route.meta.description : undefined,
)

// PLACEHOLDER DATA -- Milestone 2 ships static placeholder values, not a
// live query. Every value below is illustrative and must be replaced by
// real data once the Dashboard has something to read from.
const PLACEHOLDER_STATS = [
  { label: 'System Health', value: 'Healthy', icon: 'health', variant: 'success' },
  { label: 'Customers', value: '247', icon: 'customers', variant: 'neutral' },
  { label: 'Active Alerts', value: '0', icon: 'alert', variant: 'success' },
  { label: 'Pending Tasks', value: '2', icon: 'tasks', variant: 'neutral' },
] as const

const PLACEHOLDER_ACTIVITY = [
  { id: 'activity-1', label: 'Customer provisioned', timestamp: '2 minutes ago' },
  { id: 'activity-2', label: 'ONU registered', timestamp: '14 minutes ago' },
  { id: 'activity-3', label: 'Firmware upgrade completed', timestamp: '1 hour ago' },
  { id: 'activity-4', label: 'Inventory assigned', timestamp: '3 hours ago' },
  { id: 'activity-5', label: 'Service upgraded', timestamp: 'Yesterday' },
] as const

const PLACEHOLDER_NETWORK_OVERVIEW = [
  { label: 'OLTs', value: '1 / 1 Online' },
  { label: 'PON Ports', value: '16 / 16 Active' },
  { label: 'ONUs', value: '247 Online' },
  { label: 'Core Services', value: 'Healthy' },
]

// Two items, matching the "Pending Tasks: 2" summary card above -- see
// this milestone's summary for why the count is deliberately kept
// consistent with the card rather than the spec's three-item example.
const PLACEHOLDER_TASKS = [
  { id: 'task-1', label: 'Firmware upgrade scheduled' },
  { id: 'task-2', label: 'Provisioning awaiting approval' },
]
</script>

<template>
  <div class="dashboard-view">
    <WorkspaceHeader title="Dashboard" :subtitle="subtitle" />

    <div class="dashboard-view__stats">
      <StatisticCard
        v-for="stat in PLACEHOLDER_STATS"
        :key="stat.label"
        :label="stat.label"
        :value="stat.value"
        :icon="stat.icon"
        :variant="stat.variant"
      />
    </div>

    <div class="dashboard-view__widgets">
      <DashboardWidget title="Active Alerts" icon="alert" view-all-to="/explorer">
        <BaseEmptyState
          icon="check"
          title="No active alerts"
          description="The network is operating normally."
        />
      </DashboardWidget>

      <DashboardWidget title="Recent Activity" icon="clock" view-all-to="/explorer">
        <ol class="dashboard-view__activity">
          <li v-for="entry in PLACEHOLDER_ACTIVITY" :key="entry.id" class="dashboard-view__activity-item">
            <span class="dashboard-view__activity-label">{{ entry.label }}</span>
            <span class="dashboard-view__activity-timestamp">{{ entry.timestamp }}</span>
          </li>
        </ol>
      </DashboardWidget>

      <DashboardWidget title="Network Overview" icon="network" view-all-to="/network">
        <BasePropertyGrid :properties="PLACEHOLDER_NETWORK_OVERVIEW" />
      </DashboardWidget>

      <DashboardWidget title="Pending Tasks" icon="tasks" view-all-to="/administration">
        <ul class="dashboard-view__tasks">
          <li v-for="task in PLACEHOLDER_TASKS" :key="task.id" class="dashboard-view__task-item">
            <BaseIcon name="clock" size="sm" class="dashboard-view__task-icon" />
            <span>{{ task.label }}</span>
          </li>
        </ul>
      </DashboardWidget>
    </div>
  </div>
</template>

<style scoped>
.dashboard-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.dashboard-view__stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

.dashboard-view__widgets {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.dashboard-view__activity,
.dashboard-view__tasks {
  display: flex;
  flex-direction: column;
}

.dashboard-view__activity-item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}

.dashboard-view__activity-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.dashboard-view__activity-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.dashboard-view__activity-timestamp {
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
  white-space: nowrap;
}

.dashboard-view__task-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
}

.dashboard-view__task-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.dashboard-view__task-icon {
  color: var(--color-text-muted);
}

@media (max-width: 960px) {
  .dashboard-view__stats {
    grid-template-columns: 1fr 1fr;
  }

  .dashboard-view__widgets {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .dashboard-view__stats {
    grid-template-columns: 1fr;
  }
}
</style>
