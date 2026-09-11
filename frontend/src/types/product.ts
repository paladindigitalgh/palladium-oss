/**
 * The Product domain type (internal/product). Previously trimmed to just
 * an id and display label for the Service creation form's dropdown --
 * now carries every field the Administration Workspace's Plans panel
 * needs to create and list Products, since that is a real Product
 * Workspace in miniature (see AdministrationView.vue).
 */
export type ProductCategory = 'Internet' | 'Voice' | 'IPTV' | 'Transport' | 'ManagedWiFi' | 'Other'

/**
 * Which subscriber segment a Product is sold to -- set once, at Plan
 * creation, and never edited from a Service; Add Service uses it purely
 * to filter which Plans are selectable, it never stores its own copy
 * (see ServiceFormDialog.vue).
 */
export type ServiceType = 'Residential' | 'Business' | 'Internal'

export interface Product {
  id: string
  catalogId: string
  providerId: string
  name: string
  category: ProductCategory
  serviceType: ServiceType
  status: 'Active' | 'Retired'
}
