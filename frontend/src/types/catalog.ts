/**
 * The ProductCatalog domain type (internal/catalog), trimmed to what the
 * Plans panel's "New Plan" dialog needs -- an id and a display label, to
 * silently pick a default Catalog (see catalogRepository.ts). Not full
 * domain modeling: there is no Catalog Workspace yet.
 */
export interface Catalog {
  id: string
  name: string
}
