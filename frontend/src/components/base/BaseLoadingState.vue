<script setup lang="ts">
/**
 * docs/08-DESIGN-SYSTEM.md section 19: "Use skeleton placeholders for
 * content-heavy views." `lines` lets a caller roughly match the shape of
 * the content it's replacing without needing a bespoke skeleton per
 * workspace.
 */
withDefaults(defineProps<{ lines?: number }>(), { lines: 3 })
</script>

<template>
  <div class="base-loading-state" role="status" aria-label="Loading">
    <div v-for="line in lines" :key="line" class="base-loading-state__line" />
  </div>
</template>

<style scoped>
.base-loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.base-loading-state__line {
  height: 14px;
  border-radius: var(--radius-sm);
  background: linear-gradient(
    90deg,
    var(--color-border) 25%,
    var(--color-bg) 50%,
    var(--color-border) 75%
  );
  background-size: 200% 100%;
  animation: base-loading-shimmer 1.4s ease-in-out infinite;
}

.base-loading-state__line:last-child {
  width: 60%;
}

@keyframes base-loading-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .base-loading-state__line {
    animation: none;
  }
}
</style>
