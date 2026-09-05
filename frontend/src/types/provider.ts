/**
 * The Provider domain type (internal/provider): the retail ISP identity
 * a Plan (Product) belongs to. Irrelevant in a single-ISP deployment
 * (exactly one Provider exists, and the UI stops surfacing it once
 * that's true -- see AdministrationView.vue); real in an open-access
 * deployment with more than one ISP sharing the same physical network.
 */
export interface Provider {
  id: string
  name: string
  status: 'Active' | 'Inactive'
  description: string
}
