import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * Single source of truth for Palladium's primary navigation
 * (docs/04-NAVIGATION.md section 4, "Global Navigation should remain
 * intentionally small and stable"). Both the router (which routes exist)
 * and AppSidebar (what operators see) read from this list, so the two
 * can never drift apart.
 *
 * This is exactly docs/04-NAVIGATION.md section 4's list: Dashboard,
 * Customers, Services, Devices, Network, Inventory, Explorer,
 * Administration. An earlier milestone's NAV_ITEMS had drifted from this
 * (it still had Workflows and Plugins, and no Explorer); this list
 * corrects that rather than the other way around, since the docs were
 * already right.
 *
 * `children` is reserved for future nested navigation (Milestone 1's
 * "support future nested navigation, but do not build child menus yet")
 * -- every item leaves it undefined today, and AppSidebar does not read
 * it. It exists so a future milestone can add nested items to this list
 * without a breaking shape change.
 */
export interface NavItem {
  id: string
  label: string
  path: string
  icon: IconName
  description: string
  children?: NavItem[]
}

export const NAV_ITEMS: NavItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    path: '/dashboard',
    icon: 'dashboard',
    description: 'An overview of what needs your attention right now.',
  },
  {
    id: 'customers',
    label: 'Customers',
    path: '/customers',
    icon: 'customers',
    description: 'Search, filter, and open a customer workspace.',
  },
  {
    id: 'services',
    label: 'Services',
    path: '/services',
    icon: 'services',
    description: 'Find what is being delivered, and to whom.',
  },
  {
    id: 'devices',
    label: 'Devices',
    path: '/devices',
    icon: 'devices',
    description: 'Find managed equipment on the live network.',
  },
  {
    id: 'network',
    label: 'Network',
    path: '/network',
    icon: 'network',
    description: 'Search access networks, OLTs, and PON ports.',
  },
  {
    id: 'inventory',
    label: 'Inventory',
    path: '/inventory',
    icon: 'inventory',
    description: 'Search sites, buildings, rooms, and racks.',
  },
  {
    id: 'explorer',
    label: 'Explorer',
    path: '/explorer',
    icon: 'explorer',
    description: 'Run ad hoc queries across the OSS database.',
  },
  {
    id: 'administration',
    label: 'Administration',
    path: '/administration',
    icon: 'administration',
    description: 'Platform administration will appear here.',
  },
]
