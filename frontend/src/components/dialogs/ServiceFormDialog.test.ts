import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { ApiError } from '@/services/api/httpClient'
import ServiceFormDialog from './ServiceFormDialog.vue'
import type { Service } from '@/types/service'

/**
 * The most involved dialog: unlike every other *FormDialog, it has no
 * `{ immediate: true }` on its own `watch(() => props.open, ...)` (see
 * ServiceFormDialog.vue) -- the fetch only fires on a false->true
 * transition, exactly how the real app always uses it (a Detail
 * Workspace toggles a ref from false to true on click; it never mounts
 * this dialog pre-opened). So every test here mounts with `open: false`
 * first, then `setProps({ open: true })` to trigger it, the same as a
 * real caller would.
 */
const { createService } = vi.hoisted(() => ({ createService: vi.fn() }))
const { listProducts } = vi.hoisted(() => ({ listProducts: vi.fn() }))
const { listServiceProfiles } = vi.hoisted(() => ({ listServiceProfiles: vi.fn() }))

vi.mock('@/services/services/serviceRepository', () => ({ createService }))
vi.mock('@/services/products/productRepository', () => ({ listProducts }))
vi.mock('@/services/serviceProfiles/serviceProfileRepository', () => ({ listServiceProfiles }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

// The `open` watcher itself awaits Promise.all(...) before touching any
// state, so a single nextTick (which only flushes synchronous reactivity)
// is not enough -- three ticks reliably clears the watcher's own await,
// the resolved-mock microtask, and the DOM re-render, verified empirically.
async function settle() {
  await nextTick()
  await nextTick()
  await nextTick()
}

function newService(overrides: Partial<Service> = {}): Service {
  return {
    id: 's1',
    locationId: 'l1',
    productId: 'p1',
    serviceProfileId: 'sp1',
    status: 'Pending',
    description: '',
    activatedAt: null,
    suspendedAt: null,
    disconnectedAt: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  createService.mockReset()
  listProducts.mockReset()
  listServiceProfiles.mockReset()
})

it('preselects the first product and service profile once loaded, and submits them with the required fields', async () => {
  listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
  listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
  createService.mockResolvedValue(newService())

  const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1' } })
  await wrapper.setProps({ open: true })
  await settle()

  const productSelect = body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === 'Product')!
    .find('select')
  expect((productSelect.element as HTMLSelectElement).value).toBe('p1')

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createService).toHaveBeenCalledWith({
    locationId: 'l1',
    productId: 'p1',
    serviceProfileId: 'sp1',
    status: 'Pending',
    description: '',
  })
  expect(wrapper.emitted('created')?.[0]).toEqual([newService()])
})

it('shows "no products" and hides the submit button when no products exist yet', async () => {
  listProducts.mockResolvedValue([])
  listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])

  const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1' } })
  await wrapper.setProps({ open: true })
  await settle()

  expect(body().find('.service-form__error').text()).toContain('No products exist yet')
  expect(body().findAll('button').some((b) => b.text() === 'Add Service')).toBe(false)
})

it('surfaces the API error message on a failed submit, and does not emit created', async () => {
  listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
  listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
  createService.mockRejectedValue(new ApiError('a service already exists for this location', 'conflict', 409))

  const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1' } })
  await wrapper.setProps({ open: true })
  await settle()

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.service-form__error').text()).toBe('a service already exists for this location')
  expect(wrapper.emitted('created')).toBeUndefined()
})
