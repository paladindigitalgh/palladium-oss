<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * A single relationship summary card -- eyebrow, title, optional meta
 * line, and an action that either navigates to the related object's own
 * Detail Workspace or, when that destination doesn't exist yet, reads as
 * an honest placeholder rather than a dead link
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, "Canonical Detail Views":
 * relationships should open the same Detail Workspace regardless of
 * entry point). Extracted from Device Detail's Assignment section once
 * Service Detail's Customer section needed the identical shape
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Future Evolution").
 *
 * Renders as a real `<RouterLink>` when `to` is given (keyboard/
 * right-click/open-in-new-tab all work for free), or a plain
 * non-interactive block when it isn't -- never a link to nowhere.
 */
defineProps<{
  eyebrow: string
  title: string
  meta?: string
  to?: RouteLocationRaw
  actionLabel?: string
  placeholderLabel?: string
}>()
</script>

<template>
  <component
    :is="to ? RouterLink : 'div'"
    :to="to"
    class="relationship-card"
    :class="{ 'relationship-card--placeholder': !to }"
  >
    <span class="relationship-card__eyebrow">{{ eyebrow }}</span>
    <span class="relationship-card__title">{{ title }}</span>
    <span v-if="meta" class="relationship-card__meta">{{ meta }}</span>
    <span class="relationship-card__action" :class="{ 'relationship-card__action--disabled': !to }">
      {{ to ? (actionLabel ?? 'View') : (placeholderLabel ?? 'Not available') }}
      <BaseIcon v-if="to" name="arrow-right" size="sm" />
    </span>
  </component>
</template>

<style scoped>
.relationship-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  text-decoration: none;
  color: inherit;
  transition: border-color var(--motion-fast) var(--motion-ease);
}

a.relationship-card:hover {
  border-color: var(--color-brand);
}

a.relationship-card:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

.relationship-card--placeholder {
  border-style: dashed;
}

.relationship-card__eyebrow {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.relationship-card__title {
  font-size: var(--font-size-md);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.relationship-card__meta {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.relationship-card__action {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  margin-top: var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-brand);
}

.relationship-card__action--disabled {
  color: var(--color-text-muted);
  font-weight: var(--font-weight-regular);
}
</style>
