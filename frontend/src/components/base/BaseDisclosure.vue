<script setup lang="ts">
import { ref } from 'vue'
import BaseIcon from './BaseIcon.vue'

/**
 * A standalone "click header, reveal content below" section, for a flat
 * list of independent sibling sections (e.g. one per Provider on
 * AdministrationProvidersView.vue) -- as opposed to
 * SectionCard.vue, which is the equivalent building block for a single
 * object's Detail Workspace sections and is hard-coupled to
 * <DetailWorkspace>'s Contents-nav registry via useDetailWorkspaceContext.
 * That registry is built for "the sections of one object," not "N
 * independent things that each happen to expand," so it does not fit
 * here (and docs/09-WORKSPACE-SPECIFICATIONS.md section 16 explicitly
 * keeps Administration out of the Detail Workspace structure entirely).
 * This component reuses SectionCard's exact visual/animation language
 * (chevron rotation, CSS grid-template-rows collapse) with purely local
 * state instead.
 */
const props = withDefaults(defineProps<{ title: string; defaultOpen?: boolean }>(), { defaultOpen: false })

const open = ref(props.defaultOpen)
</script>

<template>
  <section class="base-disclosure">
    <h2 class="base-disclosure__header">
      <button type="button" class="base-disclosure__toggle" :aria-expanded="open" @click="open = !open">
        <span class="base-disclosure__title">{{ title }}</span>
        <span class="base-disclosure__extra"><slot name="extra" /></span>
        <BaseIcon
          name="chevron-down"
          size="sm"
          class="base-disclosure__chevron"
          :class="{ 'base-disclosure__chevron--collapsed': !open }"
        />
      </button>
    </h2>
    <div class="base-disclosure__collapsible" :class="{ 'base-disclosure__collapsible--collapsed': !open }">
      <div class="base-disclosure__body">
        <div class="base-disclosure__body-inner">
          <slot />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.base-disclosure {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.base-disclosure__header {
  margin: 0;
}

.base-disclosure__toggle {
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

.base-disclosure__toggle:hover {
  background-color: var(--color-bg);
}

.base-disclosure__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
}

.base-disclosure__extra {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.base-disclosure__chevron {
  margin-left: auto;
  flex-shrink: 0;
  color: var(--color-text-muted);
  transition: transform var(--motion-normal) var(--motion-ease);
}

.base-disclosure__chevron--collapsed {
  transform: rotate(-90deg);
}

.base-disclosure__collapsible {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows var(--motion-normal) var(--motion-ease);
}

.base-disclosure__collapsible--collapsed {
  grid-template-rows: 0fr;
}

.base-disclosure__body {
  /* No padding here, deliberately: this element's own box is what the
     grid-template-rows collapse above shrinks to zero height. Padding
     placed directly on it (as SectionCard.vue's equivalent .section-
     card__body does) cannot itself shrink below the padding's own size
     -- overflow:hidden clips *content* that overflows, but padding is
     never "overflow," so a collapsed track still renders at least
     padding-top + padding-bottom tall. Moving the padding one level
     down, onto a plain child with no height constraint of its own,
     lets this element's box (and therefore the whole collapsed row)
     reach a true 0. */
  overflow: hidden;
}

.base-disclosure__body-inner {
  padding: 0 var(--space-5) var(--space-5);
}

@media (prefers-reduced-motion: reduce) {
  .base-disclosure__collapsible,
  .base-disclosure__chevron {
    transition: none;
  }
}
</style>
