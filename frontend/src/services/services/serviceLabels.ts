import type { Service } from '@/types/service'
import { listProducts } from '@/services/products/productRepository'
import { listProviders } from '@/services/providers/providerRepository'

/**
 * A Service record itself is deliberately lean (see types/service.ts's
 * own doc comment) -- no name, just a ProductID and a ServiceProfileID.
 * Every place that lists Services for a human (CustomerDetailView.vue,
 * ServiceCollectionView.vue, ServiceDetailView.vue's own header) needs
 * something better than the raw id to show, so this resolves each
 * Service to "<Product name>", or "<Provider name> > <Product name>"
 * once a second Provider actually exists -- the same "only show
 * Provider once it's not the only one" rule AdministrationView.vue's
 * Plans panel and PlanFormDialog.vue already apply.
 *
 * GET /products and GET /providers each have no server-side filtering,
 * so both are fetched in full and joined client-side, the same pattern
 * every other small-reference-table lookup in this app uses -- one pair
 * of requests regardless of how many Services are being labeled.
 */
export async function resolveServiceLabels(services: Service[]): Promise<Map<string, string>> {
  const [products, providers] = await Promise.all([listProducts(), listProviders()])

  const productById = new Map(products.map((product) => [product.id, product]))
  const providerNameById = new Map(providers.map((provider) => [provider.id, provider.name]))
  const showProvider = providers.length > 1

  const labels = new Map<string, string>()
  for (const service of services) {
    const product = productById.get(service.productId)
    if (!product) {
      labels.set(service.id, service.id)
      continue
    }
    const providerName = providerNameById.get(product.providerId)
    labels.set(service.id, showProvider && providerName ? `${providerName} > ${product.name}` : product.name)
  }
  return labels
}
