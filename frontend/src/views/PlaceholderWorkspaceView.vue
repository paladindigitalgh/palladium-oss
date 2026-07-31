<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import { NAV_ITEMS } from '@/router/navigation'
import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * The placeholder shown for every NAV_ITEMS route without a real
 * implementation yet, driven entirely by its route's `meta` (see
 * src/router/index.ts) -- one shared file rather than one near-identical
 * file per still-unbuilt workspace.
 *
 * This intentionally does NOT use DetailWorkspace
 * (components/workspace/DetailWorkspace.vue): most of these routes
 * (Search Results-adjacent Explorer, Administration) are documented as
 * not being Detail Workspaces at all
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, sections 14-16), and the ones
 * that will be (Customers, Services, Devices, Network, Inventory) have
 * no sections to show yet -- there is nothing here for Contents
 * navigation to list. A plain header plus an honest empty state avoids
 * implying structure ahead of the content that will justify it.
 */
const route = useRoute()

const title = computed(() =>
  typeof route.meta.title === 'string' ? route.meta.title : String(route.name ?? 'Workspace'),
)
const description = computed(() =>
  typeof route.meta.description === 'string' ? route.meta.description : undefined,
)
const icon = computed<IconName>(
  () => NAV_ITEMS.find((item) => item.id === route.meta.navId)?.icon ?? 'inventory',
)
</script>

<template>
  <div class="placeholder-workspace-view">
    <WorkspaceHeader :title="title" :subtitle="description" />
    <BaseCard>
      <BaseEmptyState
        :icon="icon"
        title="Nothing here yet"
        description="This workspace will be built in a future milestone."
      />
    </BaseCard>
  </div>
</template>

<style scoped>
.placeholder-workspace-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}
</style>
