<script setup lang="ts">
import { useTemplateRef } from 'vue'
import ContentsNavigation from './ContentsNavigation.vue'
import { provideDetailWorkspace } from '@/composables/useDetailWorkspace'

/**
 * The Detail Workspace framework root
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure";
 * docs/11-COMPONENT-ARCHITECTURE.md, "Workspace Architecture"). Owns
 * page layout, renders Contents navigation automatically, and
 * coordinates scrolling, active-section tracking, URL fragments, and
 * section collapse state (see composables/useDetailWorkspace.ts, which
 * does the actual work -- this component is a thin template over it).
 *
 * Slot-driven, not configuration-driven: write a WorkspaceHeader and any
 * number of SectionCards as flat children --
 *
 *   <DetailWorkspace>
 *     <WorkspaceHeader title="..." subtitle="..." />
 *     <SectionCard title="Summary">...</SectionCard>
 *     <SectionCard title="Services">...</SectionCard>
 *   </DetailWorkspace>
 *
 * -- and Contents navigation is built automatically from whichever
 * SectionCards render, in the order they render, with no configuration
 * array or metadata object to keep in sync. DetailWorkspace itself knows
 * nothing about WorkspaceHeader specifically; it is simply the first
 * thing in the slot and reads as a header purely because of its own
 * typography -- "the Detail Workspace should feel like one continuous
 * operational document" (docs/09-WORKSPACE-SPECIFICATIONS.md, "Tabs"),
 * not a component with a special-cased header region.
 *
 * It is completely generic: no import here, or in any file this one
 * depends on, mentions a Customer, Service, Device, or any other
 * business object.
 */
const root = useTemplateRef<HTMLElement>('root')
provideDetailWorkspace(root)
</script>

<template>
  <div ref="root" class="detail-workspace">
    <ContentsNavigation />
    <div class="detail-workspace__content">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.detail-workspace {
  display: flex;
  align-items: flex-start;
  gap: var(--space-6);
}

.detail-workspace__content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

@media (max-width: 960px) {
  .detail-workspace {
    flex-direction: column;
    gap: var(--space-4);
  }
}
</style>
