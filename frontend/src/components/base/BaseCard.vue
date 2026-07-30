<script setup lang="ts">
withDefaults(
  defineProps<{
    padded?: boolean
    elevated?: boolean
  }>(),
  { padded: true, elevated: false },
)
</script>

<template>
  <div class="base-card" :class="{ 'base-card--padded': padded, 'base-card--elevated': elevated }">
    <slot />
  </div>
</template>

<style scoped>
.base-card {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.base-card--padded {
  padding: var(--space-5);
}

.base-card--elevated {
  box-shadow: var(--shadow-md);
}
</style>
