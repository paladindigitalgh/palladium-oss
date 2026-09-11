import type { Component } from 'vue'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { NAV_ITEMS } from './navigation'
import { useAuth } from '@/composables/useAuth'

/**
 * docs/04-NAVIGATION.md section 7: "URLs should identify operational
 * resources rather than user interface layouts." Every primary
 * navigation destination gets a plain top-level path (/customers, not
 * /views/customer-list), generated from NAV_ITEMS so the route list and
 * the sidebar can never disagree.
 *
 * Routes default to the shared PlaceholderWorkspaceView (see that file's
 * own doc comment) until a workspace has a real implementation, at which
 * point its nav id is added to VIEW_COMPONENTS below. Every primary nav
 * item is implemented now -- PlaceholderWorkspaceView still exists here
 * as the fallback a future new nav item lands on before its own view is
 * built.
 *
 * Items with NAV_ITEMS children (Administration) are excluded from this
 * generated list: they are a sidebar-only grouping with no page of their
 * own (see AppSidebar.vue and navigation.ts's own doc comments) -- their
 * children each get their own explicit route below instead, and the
 * parent's path is just a redirect to the first child.
 */
const VIEW_COMPONENTS: Record<string, () => Promise<{ default: Component }>> = {
  dashboard: () => import('@/views/DashboardView.vue'),
  customers: () => import('@/views/CustomerCollectionView.vue'),
  devices: () => import('@/views/DeviceCollectionView.vue'),
  network: () => import('@/views/NetworkCollectionView.vue'),
  explorer: () => import('@/views/ExplorerView.vue'),
}

const workspaceRoutes: RouteRecordRaw[] = NAV_ITEMS.filter((item) => !item.children).map((item) => ({
  path: item.path,
  name: item.id,
  component: VIEW_COMPONENTS[item.id] ?? (() => import('@/views/PlaceholderWorkspaceView.vue')),
  meta: {
    navId: item.id,
    title: item.label,
    description: item.description,
  },
}))

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/dashboard' },
  {
    // The one public route: everything else requires a session (see the
    // navigation guard below).
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { public: true },
  },
  ...workspaceRoutes,
  {
    // The Customer Detail Workspace (docs/09-WORKSPACE-SPECIFICATIONS.md,
    // "Navigation Flow": Customers -> Customer Collection View -> Customer
    // Detail View). Not derived from NAV_ITEMS -- it is reached by
    // selecting a row in the Customer Collection View, not from primary
    // navigation -- but it is a real, permanent route, unlike the
    // /_demo/detail-workspace route it replaces.
    path: '/customers/:id',
    name: 'customer-detail',
    component: () => import('@/views/CustomerDetailView.vue'),
  },
  {
    // The Device Detail Workspace -- reached from the Device Collection
    // View or from a Customer Detail Workspace's Devices section, same
    // pattern as /customers/:id above.
    path: '/devices/:id',
    name: 'device-detail',
    component: () => import('@/views/DeviceDetailView.vue'),
  },
  {
    // The Service Detail Workspace -- reached from the Service
    // Collection View, a Customer Detail Workspace's Services section,
    // or a Device Detail Workspace's Assignment section, same pattern as
    // /customers/:id and /devices/:id above.
    path: '/services/:id',
    name: 'service-detail',
    component: () => import('@/views/ServiceDetailView.vue'),
  },
  {
    // The Access Network Detail Workspace, root of the access-network
    // hierarchy -- reached from the Network Collection View, same
    // pattern as /customers/:id above.
    path: '/network/:id',
    name: 'access-network-detail',
    component: () => import('@/views/AccessNetworkDetailView.vue'),
  },
  {
    // The OLT Detail Workspace -- reached from an Access Network Detail
    // Workspace's OLTs section, same pattern as /network/:id above.
    path: '/network/olts/:id',
    name: 'olt-detail',
    component: () => import('@/views/OLTDetailView.vue'),
  },
  {
    // The PON Port Detail Workspace -- reached from an OLT Detail
    // Workspace's PON Ports section, same pattern as /network/olts/:id
    // above.
    path: '/network/pon-ports/:id',
    name: 'pon-port-detail',
    component: () => import('@/views/PONPortDetailView.vue'),
  },
  {
    // The Access Interface Detail Workspace -- reached from a PON Port
    // Detail Workspace's Access Interfaces section, same pattern as
    // /network/pon-ports/:id above.
    path: '/network/access-interfaces/:id',
    name: 'access-interface-detail',
    component: () => import('@/views/AccessInterfaceDetailView.vue'),
  },
  {
    // The Site Detail Workspace, root of the Inventory hierarchy --
    // reached from the Inventory Collection View, same pattern as
    // /customers/:id above. Prefixed /administration/ (2026-09-08) along
    // with its collection view, since Inventory now lives under
    // Administration in the sidebar (see navigation.ts) -- keeping the
    // detail routes under the same prefix is what lets AppSidebar's
    // isChildActive keep the Administration section expanded while
    // browsing any depth of the Inventory hierarchy.
    path: '/administration/inventory/:id',
    name: 'site-detail',
    component: () => import('@/views/SiteDetailView.vue'),
  },
  {
    // The Building Detail Workspace -- reached from a Site Detail
    // Workspace's Buildings section, same pattern as
    // /administration/inventory/:id above.
    path: '/administration/inventory/buildings/:id',
    name: 'building-detail',
    component: () => import('@/views/BuildingDetailView.vue'),
  },
  {
    // The Room Detail Workspace -- reached from a Building Detail
    // Workspace's Rooms section, same pattern as
    // /administration/inventory/buildings/:id above.
    path: '/administration/inventory/rooms/:id',
    name: 'room-detail',
    component: () => import('@/views/RoomDetailView.vue'),
  },
  {
    // The Rack Detail Workspace -- reached from a Room Detail Workspace's
    // Racks section, same pattern as /administration/inventory/rooms/:id
    // above.
    path: '/administration/inventory/racks/:id',
    name: 'rack-detail',
    component: () => import('@/views/RackDetailView.vue'),
  },
  {
    // Administration has no page of its own -- it's a sidebar-only
    // dropdown grouping its children (see navigation.ts and
    // AppSidebar.vue). A direct visit or stale bookmark still lands
    // somewhere real rather than 404ing, the same reasoning
    // { path: '/', redirect: '/dashboard' } above already establishes.
    path: '/administration',
    redirect: '/administration/providers',
  },
  {
    // The Providers page -- reached from the sidebar's Administration
    // dropdown, same "not in NAV_ITEMS' generated routes, reached by
    // clicking through" pattern as /customers/:id above. Not a Detail
    // Workspace (no single shared object, no SectionCard/DetailWorkspace)
    // -- see AdministrationProvidersView.vue's own doc comment.
    path: '/administration/providers',
    name: 'administration-providers',
    component: () => import('@/views/AdministrationProvidersView.vue'),
  },
  {
    // The Users page -- reached from the sidebar's Administration
    // dropdown, same pattern as /administration/providers above.
    path: '/administration/users',
    name: 'administration-users',
    component: () => import('@/views/AdministrationUsersView.vue'),
  },
  {
    // The Hardware page -- reached from the sidebar's Administration
    // dropdown, same pattern as /administration/providers above. Named
    // generically (not "OLT Models") since it is meant to hold other
    // physical-equipment catalogs later, not just OLT chassis types.
    path: '/administration/hardware',
    name: 'administration-hardware',
    component: () => import('@/views/AdministrationHardwareView.vue'),
  },
  {
    // The Inventory Collection View -- moved under Administration
    // (2026-09-08, at the user's explicit request, mirroring
    // Providers/Users/Hardware above) from its own former top-level nav
    // item. Its own nested detail routes below (/inventory/:id and
    // deeper) are unchanged -- only this entry point moved.
    path: '/administration/inventory',
    name: 'inventory',
    component: () => import('@/views/InventoryCollectionView.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

/**
 * Every route except /login requires a session (docs/10-IMPLEMENTATION-PLAN.md
 * section 8, "Authentication"). An unauthenticated operator navigating
 * anywhere else is redirected to /login; an already-authenticated
 * operator visiting /login is sent to the dashboard instead of being
 * shown the form again.
 */
router.beforeEach((to) => {
  const { isAuthenticated } = useAuth()

  if (!to.meta.public && !isAuthenticated.value) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.public && isAuthenticated.value) {
    return { name: 'dashboard' }
  }
  return true
})
