<script setup lang="ts">
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="not-found">
    <BaseEmptyState
      icon="search"
      title="Page not found"
      description="There is no workspace at this address."
    >
      <RouterLink to="/dashboard" custom v-slot="{ navigate }">
        <BaseButton variant="primary" @click="navigate">Go to Dashboard</BaseButton>
      </RouterLink>
    </BaseEmptyState>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
}
</style>
