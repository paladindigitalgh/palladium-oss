/**
 * The ProvisioningProfile domain type (internal/provisioning) -- maps one
 * Product to the named configuration profile a specific OLT vendor
 * already has configured for it (built by an operator directly on the
 * OLT; see internal/provisioning's own package doc comment). Palladium
 * never generates or modifies the profile itself.
 */
export interface ProvisioningProfile {
  id: string
  productId: string
  vendor: string
  profileName: string
  description: string
}
