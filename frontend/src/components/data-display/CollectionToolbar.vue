<script setup lang="ts">
import BaseIcon from '@/components/base/BaseIcon.vue'

/**
 * The search-plus-filters bar every Collection View opens with
 * (docs/09-WORKSPACE-SPECIFICATIONS.md, "Collection View": "Support
 * search," "Support filtering"). Owns only layout and the search input;
 * filter controls (BaseSelect instances, one per filter) are supplied
 * through the default slot so this component stays generic across
 * Customers, Devices, Services, and every future Collection Workspace --
 * it has no idea what it is filtering.
 */
defineProps<{
  searchPlaceholder?: string
}>()

const search = defineModel<string>('search', { required: true })
</script>

<template>
  <div class="collection-toolbar">
    <label class="collection-toolbar__search">
      <span class="collection-toolbar__search-label">Search</span>
      <span class="collection-toolbar__search-control">
        <BaseIcon name="search" size="sm" class="collection-toolbar__search-icon" />
        <input
          v-model="search"
          type="search"
          class="collection-toolbar__search-input"
          :placeholder="searchPlaceholder ?? 'Search'"
        />
        <button
          v-if="search"
          type="button"
          class="collection-toolbar__search-clear"
          aria-label="Clear search"
          @click="search = ''"
        >
          <BaseIcon name="close" size="sm" />
        </button>
      </span>
    </label>

    <div v-if="$slots.default" class="collection-toolbar__filters">
      <span class="collection-toolbar__filters-label">
        <BaseIcon name="filter" size="sm" />
        Filters
      </span>
      <slot />
    </div>
  </div>
</template>

<style scoped>
.collection-toolbar {
  display: flex;
  align-items: flex-end;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.collection-toolbar__search {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  flex: 1 1 260px;
  min-width: 220px;
}

.collection-toolbar__search-label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.collection-toolbar__search-control {
  position: relative;
  display: flex;
  align-items: center;
}

.collection-toolbar__search-icon {
  position: absolute;
  left: var(--space-3);
  color: var(--color-text-muted);
  pointer-events: none;
}

.collection-toolbar__search-input {
  width: 100%;
  padding: var(--space-2) var(--space-7) var(--space-2) var(--space-7);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background-color: var(--color-surface);
  color: var(--color-text-primary);
  font: inherit;
  font-size: var(--font-size-sm);
}

.collection-toolbar__search-input:hover {
  border-color: var(--color-text-muted);
}

.collection-toolbar__search-input::-webkit-search-cancel-button {
  display: none;
}

.collection-toolbar__search-clear {
  position: absolute;
  right: var(--space-2);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--color-text-muted);
  cursor: pointer;
}

.collection-toolbar__search-clear:hover {
  color: var(--color-text-primary);
  background-color: var(--color-bg);
}

.collection-toolbar__filters {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.collection-toolbar__filters-label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding-bottom: var(--space-2);
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
