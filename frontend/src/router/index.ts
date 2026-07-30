import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { NAV_ITEMS } from './navigation'

/**
 * docs/04-NAVIGATION.md section 7: "URLs should identify operational
 * resources rather than user interface layouts." Every primary
 * navigation destination gets a plain top-level path (/customers, not
 * /views/customer-list), generated from NAV_ITEMS so the route list and
 * the sidebar can never disagree.
 *
 * Every route renders the same PlaceholderWorkspaceView component (see
 * that file's own doc comment) -- this milestone builds no business
 * functionality, so there is nothing yet to differentiate one route's
 * component from another's.
 */
const workspaceRoutes: RouteRecordRaw[] = NAV_ITEMS.map((item) => ({
  path: item.path,
  name: item.id,
  component: () => import('@/views/PlaceholderWorkspaceView.vue'),
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
