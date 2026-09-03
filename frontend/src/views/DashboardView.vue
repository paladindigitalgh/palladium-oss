<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import StatisticCard from '@/components/data-display/StatisticCard.vue'
import DashboardWidget from '@/components/dashboard/DashboardWidget.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BasePropertyGrid from '@/components/base/BasePropertyGrid.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import BaseErrorState from '@/components/base/BaseErrorState.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import ActivityList from '@/components/data-display/ActivityList.vue'
import { useDashboard } from '@/composables/useDashboard'
import { formatDisplayDate as formatDate } from '@/lib/dates'
import type { IconName } from '@/components/base/BaseIcon.vue'

interface StatCard {
  label: string
  value: string
  icon: IconName
  variant: 'success' | 'warning' | 'error' | 'info' | 'neutral'
}

/**
 * Backed by the real API via useDashboard.ts -- every stat and widget
 * below is a live query, not the placeholder data this view shipped
 * with through Milestone 2. Two things from the original speculative
 * design are deliberately absent, not just unfinished:
 *
 * - "Active Alerts" (a stat card and a widget): no alert concept exists
 *   anywhere in the domain model, and per CLAUDE.md, Palladium is not a
 *   monitoring platform -- the same reason Alarms/Performance sections
 *   were already dropped from the Device and Service Detail Workspaces.
 * - "System Health": /healthz and /readyz exist, but are mounted outside
 *   /api/v1 (unauthenticated, a different base path than every other
 *   endpoint this frontend calls through apiFetch). Wiring one stat to a
 *   separate fetch mechanism isn't worth what it buys; a fourth real
 *   domain stat (Devices) takes its place instead.
 *
 * "Network Overview" no longer shows online/active ratios ("16/16
 * Active") -- that is telemetry Palladium does not have. It shows real
 * counts instead, including the one genuinely real status split
 * available at this level: Access Interfaces by Active/Disabled, an
 * administrative field, not live telemetry.
 */
const route = useRoute()
const subtitle = computed(() =>
  typeof route.meta.description === 'string' ? route.meta.description : undefined,
)

const { loading, error, stats, networkOverview, recentActivity, pendingTasks, retry } = useDashboard()

const statCards = computed<StatCard[]>(() => [
  { label: 'Customers', value: String(stats.value.customers), icon: 'customers', variant: 'neutral' },
  { label: 'Active Services', value: String(stats.value.activeServices), icon: 'services', variant: 'neutral' },
  { label: 'Devices', value: String(stats.value.devices), icon: 'devices', variant: 'neutral' },
  {
    label: 'Pending Tasks',
    value: String(stats.value.pendingTasks),
    icon: 'tasks',
    // docs/08-DESIGN-SYSTEM.md section 3: "calm under normal operation,
    // visually emphatic only when action is required" -- zero pending
    // tasks is the calm state, any at all is the one worth a glance.
    variant: stats.value.pendingTasks > 0 ? 'warning' : 'success',
  },
])

const activityEntries = computed(() =>
  recentActivity.value.map((event) => ({ id: event.id, label: event.message, timestamp: formatDate(event.createdAt) })),
)

const networkProperties = computed(() => [
  { label: 'Access Networks', value: String(networkOverview.value.accessNetworks) },
  { label: 'OLTs', value: String(networkOverview.value.olts) },
  { label: 'PON Ports', value: String(networkOverview.value.ponPorts) },
  { label: 'Access Interfaces (Active)', value: String(networkOverview.value.activeInterfaces) },
  { label: 'Access Interfaces (Disabled)', value: String(networkOverview.value.disabledInterfaces) },
])
</script>

<template>
  <div v-if="loading" class="dashboard-view__status">
    <BaseLoadingState :lines="8" />
  </div>

  <div v-else-if="error" class="dashboard-view__status">
    <BaseErrorState title="Dashboard could not be loaded" description="Something went wrong fetching the latest data.">
      <BaseButton variant="secondary" @click="retry">Retry</BaseButton>
    </BaseErrorState>
  </div>

  <div v-else class="dashboard-view">
    <WorkspaceHeader title="Dashboard" :subtitle="subtitle" />

    <div class="dashboard-view__stats">
      <StatisticCard
        v-for="stat in statCards"
        :key="stat.label"
        :label="stat.label"
        :value="stat.value"
        :icon="stat.icon"
        :variant="stat.variant"
      />
    </div>

    <div class="dashboard-view__widgets">
      <DashboardWidget title="Recent Activity" icon="clock" view-all-to="/explorer">
        <ActivityList :entries="activityEntries" />
      </DashboardWidget>

      <DashboardWidget title="Network Overview" icon="network" view-all-to="/network">
        <BasePropertyGrid :properties="networkProperties" />
      </DashboardWidget>

      <DashboardWidget title="Pending Tasks" icon="tasks" view-all-to="/administration">
        <BaseEmptyState
          v-if="pendingTasks.length === 0"
          icon="check"
          title="Nothing pending"
          description="Every workflow has run to completion."
        />
        <ul v-else class="dashboard-view__tasks">
          <li v-for="task in pendingTasks" :key="task.id" class="dashboard-view__task-item">
            <RouterLink :to="`/services/${task.serviceId}`" class="dashboard-view__task-link">
              {{ task.definitionName }} — {{ task.status }}
            </RouterLink>
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

.dashboard-view__status {
  padding: var(--space-6);
}

.dashboard-view__stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

.dashboard-view__widgets {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-4);
}

.dashboard-view__tasks {
  display: flex;
  flex-direction: column;
}

.dashboard-view__task-item {
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
  font-size: var(--font-size-sm);
}

.dashboard-view__task-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.dashboard-view__task-link {
  color: var(--color-text-primary);
  text-decoration: none;
}

.dashboard-view__task-link:hover {
  color: var(--color-brand);
}

@media (max-width: 1200px) {
  .dashboard-view__widgets {
    grid-template-columns: 1fr 1fr;
  }
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
