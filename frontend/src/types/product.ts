/**
 * The Product domain type (internal/product). Previously trimmed to just
 * an id and display label for the Service creation form's dropdown --
 * now carries every field the Administration Workspace's Plans panel
 * needs to create and list Products, since that is a real Product
 * Workspace in miniature (see AdministrationView.vue).
 */
export type ProductCategory = 'Internet' | 'Voice' | 'IPTV' | 'Transport' | 'ManagedWiFi' | 'Other'

export interface Product {
  id: string
  catalogId: string
  providerId: string
  name: string
  category: ProductCategory
  status: 'Active' | 'Retired'
  description: string
}
