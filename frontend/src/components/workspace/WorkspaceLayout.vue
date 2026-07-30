<script setup lang="ts">
/**
 * Arranges the standard Workspace layout regions (docs/07-UI-
 * ARCHITECTURE.md section 12: Header, Primary Content, Secondary
 * Information, Right Sidebar, Timeline) into one consistent grid, so
 * "every workspace uses the shared Workspace layout"
 * (docs/11-COMPONENT-ARCHITECTURE.md, Architectural Rules #1) without
 * each domain reinventing the arrangement.
 *
 * Purely a grid -- it takes no props and holds no state; every region is
 * a named slot so the composing view supplies its own WorkspaceHeader,
 * WorkspaceSummary, WorkspaceContent, RelationshipPanel, and
 * TimelinePanel instances.
 */
</script>

<template>
  <div class="workspace-layout">
    <div class="workspace-layout__header">
      <slot name="header" />
    </div>
    <div class="workspace-layout__main">
      <slot name="summary" />
      <slot name="content" />
    </div>
    <div class="workspace-layout__sidebar">
      <slot name="relationships" />
      <slot name="timeline" />
    </div>
  </div>
</template>

<style scoped>
.workspace-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  grid-template-areas: 'header header' 'main sidebar';
  gap: var(--space-5);
  align-items: start;
}

.workspace-layout__header {
  grid-area: header;
}

.workspace-layout__main {
  grid-area: main;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}

.workspace-layout__sidebar {
  grid-area: sidebar;
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

@media (max-width: 960px) {
  .workspace-layout {
    grid-template-columns: 1fr;
    grid-template-areas: 'header' 'main' 'sidebar';
  }
}
</style>
