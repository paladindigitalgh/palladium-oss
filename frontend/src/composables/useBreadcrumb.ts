import { ref, computed } from 'vue'
import type { RouteLocationRaw } from 'vue-router'

export interface BreadcrumbItem {
  label: string
  to?: RouteLocationRaw
}

/**
 * The shell-level directory trail (TopNavigation.vue), App-owned state
 * shared by every component -- the same module-scope-singleton pattern
 * useAuth.ts/useTheme.ts/useSidebar.ts already use for their own
 * App-owned state.
 *
 * The router's global beforeEach guard (router/index.ts) calls
 * setBreadcrumb with each route's static meta.breadcrumb on every
 * navigation, before the new view even mounts -- this is what most pages
 * need and all a Collection View or a parent-less root Detail View
 * (Customer, Device, OLT, Site) ever sees. A handful of Detail views
 * whose immediate parent is only known after an async fetch (Service,
 * PON Port, Access Interface, Building, Room, Rack) call setBreadcrumb a
 * second time once that fetch resolves, upgrading the static fallback to
 * include the real parent record's name and a link to it -- see each of
 * those views' own load() for the exact call.
 */
const items = ref<BreadcrumbItem[]>([])

export function setBreadcrumb(next: BreadcrumbItem[]): void {
  items.value = next
}

export function useBreadcrumb() {
  return { items: computed(() => items.value) }
}
