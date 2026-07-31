<script setup lang="ts">
import BaseCard from '@/components/base/BaseCard.vue'
import BaseIcon, { type IconName } from '@/components/base/BaseIcon.vue'

/**
 * StatisticCard (docs/11-COMPONENT-ARCHITECTURE.md, "Data Display" shared
 * components: "StatisticCard"). A single at-a-glance metric: an icon, a
 * label, and a value. Generic over both -- it has no notion of what
 * "System Health" or "Customers" means, so any future workspace can use
 * it for its own top-line metrics, not just the Dashboard.
 *
 * `variant` tints only the icon badge, not the value text.
 * docs/08-DESIGN-SYSTEM.md section 3: "Calm under normal operation.
 * Visually emphatic only when action is required." A wall of
 * large colored numbers reads as noisy; the value stays a plain,
 * readable --color-text-primary regardless of variant, and the icon
 * badge is where status is allowed to show through.
 */
withDefaults(
  defineProps<{
    label: string
    value: string
    icon: IconName
    variant?: 'success' | 'warning' | 'error' | 'info' | 'neutral'
  }>(),
  { variant: 'neutral' },
)
</script>

<template>
  <BaseCard class="statistic-card">
    <div class="statistic-card__icon" :class="`statistic-card__icon--${variant}`">
      <BaseIcon :name="icon" />
    </div>
    <dl class="statistic-card__body">
      <dt class="statistic-card__label">{{ label }}</dt>
      <dd class="statistic-card__value">{{ value }}</dd>
    </dl>
  </BaseCard>
</template>

<style scoped>
.statistic-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.statistic-card__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.statistic-card__icon--neutral {
  background-color: var(--color-bg);
  color: var(--color-text-secondary);
}

.statistic-card__icon--success {
  background-color: var(--color-success-bg);
  color: var(--color-success);
}

.statistic-card__icon--warning {
  background-color: var(--color-warning-bg);
  color: var(--color-warning);
}

.statistic-card__icon--error {
  background-color: var(--color-error-bg);
  color: var(--color-error);
}

.statistic-card__icon--info {
  background-color: var(--color-info-bg);
  color: var(--color-info);
}

.statistic-card__body {
  margin: 0;
  min-width: 0;
}

.statistic-card__label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.statistic-card__value {
  margin: var(--space-1) 0 0;
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
  line-height: var(--line-height-tight);
}
</style>
