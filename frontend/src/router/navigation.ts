import type { IconName } from '@/components/base/BaseIcon.vue'

/**
 * Single source of truth for Palladium's primary navigation
 * (docs/04-NAVIGATION.md section 4, "Global Navigation should remain
 * intentionally small and stable"). Both the router (which routes exist)
 * and AppSidebar (what operators see) read from this list, so the two
 * can never drift apart.
 *
 * This is docs/04-NAVIGATION.md section 4's list: Dashboard, Customers,
 * Devices, Network, Explorer, Administration. An earlier milestone's
 * NAV_ITEMS had drifted from this (it still had Workflows and Plugins,
 * and no Explorer); this list corrects that rather than the other way
 * around, since the docs were already right.
 *
 * Services and Inventory are deliberately not top-level items. Services
 * has no Collection View of its own at all (2026-09-08, at the user's
 * explicit request: "my workflow will always be search for customer and
 * see their service, never search for a service directly") -- a Service
 * is only ever reached through a Customer's Detail Workspace, or a
 * Device's Assignment section, never browsed on its own; ServiceDetailView
 * and its /services/:id route are unaffected, only the collection page
 * and nav entry are gone. Inventory still has a real Collection View, but
 * lives as a child under Administration instead (see below) -- also the
 * user's explicit request, mirroring how Providers/Users/Hardware are
 * grouped there.
 *
 * `children` was reserved for future nested navigation (Milestone 1's
 * "support future nested navigation, but do not build child menus yet")
 * and is now real: Administration was the first item to use it (a
 * default-collapsed dropdown in AppSidebar.vue, not a page of its own --
 * see that component's own doc comment), at the user's explicit request.
 * Explorer (2026-09-11, also the user's explicit request) followed the
 * same pattern, splitting into Reports and Activity. Every other item
 * still leaves it undefined.
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
    description: 'Search OLTs, PON ports, and access interfaces.',
  },
  {
    // No page of its own -- /explorer redirects to the first child (see
    // router/index.ts) -- this item exists purely to group its children
    // under one collapsible sidebar entry, the same pattern Administration
    // (below) already established.
    id: 'explorer',
    label: 'Explorer',
    path: '/explorer',
    icon: 'explorer',
    description: 'Pull data out of the OSS: reports and activity history.',
    children: [
      {
        id: 'explorer-reports',
        label: 'Reports',
        path: '/explorer/reports',
        icon: 'explorer',
        description: 'Curated cross-domain reports, searchable and exportable to CSV.',
      },
      {
        id: 'explorer-activity',
        label: 'Activity',
        path: '/explorer/activity',
        icon: 'history',
        description: 'A searchable history of everything that has happened in the OSS.',
      },
    ],
  },
  {
    id: 'administration',
    label: 'Administration',
    path: '/administration',
    icon: 'administration',
    description: 'Manage Plans and other platform configuration.',
    // No page of its own -- /administration redirects to the first
    // child (see router/index.ts) -- this item exists purely to group
    // its children under one collapsible sidebar entry.
    children: [
      {
        id: 'administration-providers',
        label: 'Providers',
        path: '/administration/providers',
        icon: 'settings',
        description: 'Retail ISP identities and the Plans each one sells.',
      },
      {
        id: 'administration-users',
        label: 'Users',
        path: '/administration/users',
        icon: 'user',
        description: 'Platform login accounts and their roles.',
      },
      {
        id: 'administration-hardware',
        label: 'Hardware',
        path: '/administration/hardware',
        icon: 'settings',
        description: 'Physical equipment catalogs, starting with OLT chassis types and their PON port counts.',
      },
      {
        id: 'administration-inventory',
        label: 'Inventory',
        path: '/administration/inventory',
        icon: 'inventory',
        description: 'Search sites, buildings, rooms, and racks.',
      },
    ],
  },
]
