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
 * point its nav id is added to VIEW_COMPONENTS below. Dashboard,
 * Customers, Devices, and Services are implemented; the rest will follow
 * the same pattern rather than each needing its own routing logic.
 */
const VIEW_COMPONENTS: Record<string, () => Promise<{ default: Component }>> = {
  dashboard: () => import('@/views/DashboardView.vue'),
  customers: () => import('@/views/CustomerCollectionView.vue'),
  devices: () => import('@/views/DeviceCollectionView.vue'),
  services: () => import('@/views/ServiceCollectionView.vue'),
  network: () => import('@/views/NetworkCollectionView.vue'),
}

const workspaceRoutes: RouteRecordRaw[] = NAV_ITEMS.map((item) => ({
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
