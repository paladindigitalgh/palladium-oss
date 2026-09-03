/**
 * The Customer domain type, matching internal/customer's real API shape
 * exactly (internal/customer/httpapi/dto.go's customerResponse) --
 * unlike this file's previous mock version, nothing here is fabricated
 * to fill out a richer UI. Per docs/03-DOMAIN-MODEL.md section 4, a
 * Customer is an identity record only: no address (see types/location.ts
 * for that), no embedded services, no contacts, no alerts.
 */
export type CustomerType = 'Residential' | 'Business' | 'Government' | 'Internal'

export type CustomerStatus = 'Active' | 'Inactive' | 'Archived'

export interface Customer {
  id: string
  name: string
  customerType: CustomerType
  status: CustomerStatus
  description: string
  createdAt: string
  updatedAt: string
}
