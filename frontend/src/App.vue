<script setup lang="ts">
import { useRoute } from 'vue-router'
import AppShell from '@/components/app/AppShell.vue'

/**
 * The one public route (/login, see router/index.ts) renders standalone,
 * without AppShell's sidebar, top navigation, or search -- there is no
 * session yet for any of that global chrome to act on. Every other
 * route renders inside AppShell as before.
 */
const route = useRoute()
</script>

<template>
  <RouterView v-if="route.meta.public" />
  <AppShell v-else />
</template>
