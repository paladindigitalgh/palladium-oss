import type { Component } from 'vue'
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { NAV_ITEMS } from './navigation'

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
