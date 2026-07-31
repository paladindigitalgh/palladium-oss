<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, useId, useTemplateRef } from 'vue'
import BaseIcon, { type IconName } from '@/components/base/BaseIcon.vue'
import BaseBadge from '@/components/base/BaseBadge.vue'
import { slugifySectionTitle, useDetailWorkspaceContext } from '@/composables/useDetailWorkspace'

/**
 * SectionCard (docs/08-DESIGN-SYSTEM.md, "SectionCard"): the standard
 * building block for every Detail Workspace section
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, "Detail Workspace Structure").
 * Used directly inside <DetailWorkspace> -- no separate registration
 * step: mounting registers this section with the workspace's Contents
 * navigation, unmounting removes it.
 *
 * `id` is optional -- when omitted, it is derived from `title`
 * ("Active Alerts" -> "active-alerts") so the common case needs no extra
 * prop. Pass an explicit `id` when a title might change independently of
 * the URL fragment it should keep.
 */
const props = defineProps<{
  title: string
  id?: string
  icon?: IconName
  badge?: string | number
}>()

const registry = useDetailWorkspaceContext()
const root = useTemplateRef<HTMLElement>('root')
const headingId = `section-card-heading-${useId()}`
const bodyId = `section-card-body-${useId()}`

const sectionId = computed(() => props.id ?? slugifySectionTitle(props.title))
const collapsed = computed(() => registry.isCollapsed(sectionId.value))

function toggle() {
  registry.toggleCollapsed(sectionId.value)
}

onMounted(() => {
  if (root.value) {
    registry.registerSection({ id: sectionId.value, title: props.title, icon: props.icon }, root.value)
  }
})

onBeforeUnmount(() => {
  registry.unregisterSection(sectionId.value)
})
</script>

<template>
  <section :id="sectionId" ref="root" class="section-card" tabindex="-1">
    <h2 class="section-card__header">
      <button
        :id="headingId"
        type="button"
        class="section-card__toggle"
        :aria-expanded="!collapsed"
        :aria-controls="bodyId"
        @click="toggle"
      >
        <BaseIcon v-if="icon" :name="icon" size="sm" class="section-card__icon" />
        <span class="section-card__title">{{ title }}</span>
        <BaseBadge v-if="badge !== undefined" class="section-card__badge">{{ badge }}</BaseBadge>
        <BaseIcon
          name="chevron-down"
          size="sm"
          class="section-card__chevron"
          :class="{ 'section-card__chevron--collapsed': collapsed }"
        />
      </button>
    </h2>
    <div class="section-card__collapsible" :class="{ 'section-card__collapsible--collapsed': collapsed }">
      <div :id="bodyId" class="section-card__body" role="region" :aria-labelledby="headingId">
        <slot />
      </div>
    </div>
  </section>
</template>

<style scoped>
.section-card {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  /* Keeps a scrolled-to section from tucking under the sticky app top
     bar; must stay roughly in sync with DETAIL_WORKSPACE_SCROLL_OFFSET
     in composables/useDetailWorkspace.ts. */
  scroll-margin-top: 88px;
}

.section-card__header {
  margin: 0;
}

.section-card__toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-4) var(--space-5);
  border: none;
  background: transparent;
  color: var(--color-text-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.section-card__toggle:hover {
  background-color: var(--color-bg);
}

.section-card__icon {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.section-card__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
}

.section-card__badge {
  flex-shrink: 0;
}

.section-card__chevron {
  margin-left: auto;
  flex-shrink: 0;
  color: var(--color-text-muted);
  transition: transform var(--motion-normal) var(--motion-ease);
}

.section-card__chevron--collapsed {
  transform: rotate(-90deg);
}

.section-card__collapsible {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows var(--motion-normal) var(--motion-ease);
}

.section-card__collapsible--collapsed {
  grid-template-rows: 0fr;
}

.section-card__body {
  overflow: hidden;
  padding: 0 var(--space-5) var(--space-5);
}

@media (prefers-reduced-motion: reduce) {
  .section-card__collapsible,
  .section-card__chevron {
    transition: none;
  }
}
</style>
