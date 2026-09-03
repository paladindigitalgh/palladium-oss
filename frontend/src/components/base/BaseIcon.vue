<script setup lang="ts">
import { computed, type Component } from 'vue'
import {
  LayoutDashboard,
  Users,
  Layers,
  Server,
  Network,
  Package,
  Telescope,
  Settings,
  Search,
  Bell,
  User,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  Menu,
  Sun,
  Moon,
  X,
  Clock,
  ArrowRight,
  TriangleAlert,
  Activity,
  ListTodo,
  CircleCheck,
  ListFilter,
  MapPin,
  ChevronsUpDown,
  History,
  StickyNote,
  Settings2,
} from '@lucide/vue'

/**
 * Palladium's icon strategy (architecture decision): Lucide is the
 * official icon library, but BaseIcon is the ONLY component permitted to
 * import it. Every other component renders icons exclusively through
 * BaseIcon's `name` prop -- a plain string union with no Lucide type or
 * component ever exposed to a caller. Swapping icon libraries in the
 * future means changing the import above and the ICONS map below; it
 * should never require touching a consumer.
 *
 * IconName is intentionally a closed set (not `string`): an unrecognized
 * name is a compile error, not a silently blank icon, and it keeps
 * exactly one icon per concept across the app (docs/08-DESIGN-SYSTEM.md
 * section 9: "consistency is more important than icon variety").
 *
 * Icons default to decorative (aria-hidden="true"): a bare glyph should
 * never be a control's only label -- label the control (aria-label),
 * not the glyph. A caller that genuinely needs the icon itself announced
 * can still override this by passing its own aria-hidden/aria-label,
 * since both are forwarded through as fallthrough attributes.
 */
export type IconName =
  | 'dashboard'
  | 'customers'
  | 'services'
  | 'devices'
  | 'network'
  | 'inventory'
  | 'explorer'
  | 'administration'
  | 'search'
  | 'bell'
  | 'user'
  | 'chevron-left'
  | 'chevron-right'
  | 'chevron-down'
  | 'menu'
  | 'sun'
  | 'moon'
  | 'close'
  | 'clock'
  | 'arrow-right'
  | 'alert'
  | 'health'
  | 'tasks'
  | 'check'
  | 'filter'
  | 'location'
  | 'sort'
  | 'history'
  | 'notes'
  | 'settings'

// Typed as Vue's generic `Component`, not Lucide's own icon type -- even
// this internal map stays library-agnostic, so nothing here nudges a
// future maintainer toward leaking Lucide types further than necessary.
const ICONS: Record<IconName, Component> = {
  dashboard: LayoutDashboard,
  customers: Users,
  services: Layers,
  devices: Server,
  network: Network,
  inventory: Package,
  explorer: Telescope,
  administration: Settings,
  search: Search,
  bell: Bell,
  user: User,
  'chevron-left': ChevronLeft,
  'chevron-right': ChevronRight,
  'chevron-down': ChevronDown,
  menu: Menu,
  sun: Sun,
  moon: Moon,
  close: X,
  clock: Clock,
  'arrow-right': ArrowRight,
  alert: TriangleAlert,
  health: Activity,
  tasks: ListTodo,
  check: CircleCheck,
  filter: ListFilter,
  location: MapPin,
  sort: ChevronsUpDown,
  history: History,
  notes: StickyNote,
  settings: Settings2,
}

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    name: IconName
    size?: 'sm' | 'md' | 'lg'
    /** Forwarded to Lucide's own strokeWidth prop; omit to use its default. */
    strokeWidth?: number
  }>(),
  { size: 'md' },
)

const icon = computed(() => ICONS[props.name])
</script>

<template>
  <component
    :is="icon"
    class="base-icon"
    :class="`base-icon--${size}`"
    :stroke-width="strokeWidth"
    aria-hidden="true"
    v-bind="$attrs"
  />
</template>

<style scoped>
/*
 * Color is deliberately not a prop: every Lucide icon strokes with
 * currentColor, so color comes from the ordinary CSS `color` an
 * ancestor sets -- the same mechanism the rest of the app already uses
 * for text color, not a bespoke icon API.
 */
.base-icon {
  flex-shrink: 0;
}

.base-icon--sm {
  width: var(--icon-size-sm);
  height: var(--icon-size-sm);
}

.base-icon--md {
  width: var(--icon-size-md);
  height: var(--icon-size-md);
}

.base-icon--lg {
  width: var(--icon-size-lg);
  height: var(--icon-size-lg);
}
</style>
