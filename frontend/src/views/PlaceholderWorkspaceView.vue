<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import WorkspaceLayout from '@/components/workspace/WorkspaceLayout.vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceContent from '@/components/workspace/WorkspaceContent.vue'
import RelationshipPanel from '@/components/workspace/RelationshipPanel.vue'
import TimelinePanel from '@/components/workspace/TimelinePanel.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import { NAV_ITEMS } from '@/router/navigation'
import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * The one workspace view Milestone 1 ships. Every NAV_ITEMS route
 * renders this single component, driven entirely by its route's `meta`
 * (see src/router/index.ts) -- eight near-identical files would violate
 * "do not duplicate layout code."
 *
 * Content is deliberately an honest empty state, not fabricated example
 * data: Milestone 1 asks for "page title, short placeholder description,
 * empty-state layout appropriate for future development," not a
 * populated-looking mockup. An earlier version of this component filled
 * the summary/relationships/timeline regions with labelled "example"
 * rows to demonstrate the primitives render correctly; that was useful
 * for reviewing the layout once, but reads as noise in a shell meant to
 * feel production-quality, so this version leans on RelationshipPanel's
 * and TimelinePanel's own built-in empty states instead of feeding them
 * fake rows.
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
  <WorkspaceLayout>
    <template #header>
      <WorkspaceHeader :title="title" :subtitle="description" />
    </template>

    <template #content>
      <WorkspaceContent>
        <BaseEmptyState
          :icon="icon"
          title="Nothing here yet"
          description="This workspace will be built in a future milestone."
        />
      </WorkspaceContent>
    </template>

    <template #relationships>
      <RelationshipPanel />
    </template>

    <template #timeline>
      <TimelinePanel />
    </template>
  </WorkspaceLayout>
</template>
