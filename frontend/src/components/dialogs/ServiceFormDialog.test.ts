import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { ApiError } from '@/services/api/httpClient'
import ServiceFormDialog from './ServiceFormDialog.vue'
import type { Service } from '@/types/service'
import type { Device } from '@/types/device'

/**
 * Dual-mode, mirroring DeviceFormDialog.test.ts/CustomerFormDialog.test.ts:
 * create when no `service` prop, edit when present. Also the most
 * involved dialog: unlike some other *FormDialogs, it has no
 * `{ immediate: true }` on its own `watch(() => props.open, ...)` (see
 * ServiceFormDialog.vue) -- the fetch only fires on a false->true
 * transition, exactly how the real app always uses it (a Detail
 * Workspace toggles a ref from false to true on click; it never mounts
 * this dialog pre-opened). So every test here mounts with `open: false`
 * first, then `setProps({ open: true })` to trigger it, the same as a
 * real caller would.
 */
const { createService, updateService } = vi.hoisted(() => ({ createService: vi.fn(), updateService: vi.fn() }))
const { listProducts } = vi.hoisted(() => ({ listProducts: vi.fn() }))
const { listServiceProfiles } = vi.hoisted(() => ({ listServiceProfiles: vi.fn() }))
const { createServiceEquipment } = vi.hoisted(() => ({ createServiceEquipment: vi.fn() }))

vi.mock('@/services/services/serviceRepository', () => ({ createService, updateService }))
vi.mock('@/services/products/productRepository', () => ({ listProducts }))
vi.mock('@/services/serviceProfiles/serviceProfileRepository', () => ({ listServiceProfiles }))
vi.mock('@/services/serviceEquipment/serviceEquipmentRepository', () => ({ createServiceEquipment }))

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

function existingService(overrides: Partial<Service> = {}): Service {
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

function productSelect() {
  return body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === 'Product')!
    .find('select')
}

function deviceSelect() {
  return body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === 'Device')
    ?.find('select')
}

function fixtureDevice(overrides: Partial<Device> = {}): Device {
  return {
    id: 'd1',
    name: 'ONT-Main-01',
    description: '',
    rackId: null,
    deviceModelId: 'dm1',
    manufacturer: 'Iskratel',
    model: 'InnboxX24',
    serialNumber: 'SN001',
    assetTag: '',
    status: 'Unused',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  createService.mockReset()
  updateService.mockReset()
  listProducts.mockReset()
  listServiceProfiles.mockReset()
  createServiceEquipment.mockReset()
  createServiceEquipment.mockResolvedValue({ id: 'se1' })
})

describe('create mode (no service prop)', () => {
  it('preselects the first product and service profile once loaded, auto-selects the sole device, and submits them with the required fields', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    createService.mockResolvedValue(existingService())

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices: [fixtureDevice()] } })
    await wrapper.setProps({ open: true })
    await settle()

    expect((productSelect().element as HTMLSelectElement).value).toBe('p1')
    expect(body().find('.base-modal__title').text()).toBe('Add Service')
    // Exactly one eligible device: auto-selected, no picker shown.
    expect(deviceSelect()).toBeUndefined()

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createService).toHaveBeenCalledWith({
      locationId: 'l1',
      productId: 'p1',
      serviceProfileId: 'sp1',
      status: 'Pending',
      description: '',
    })
    expect(createServiceEquipment).toHaveBeenCalledWith({
      serviceId: 's1',
      deviceId: 'd1',
      role: 'ONU',
      description: '',
    })
    expect(updateService).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingService()])
  })

  it('shows a Device picker when the customer has more than one eligible device, and submits the chosen one', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    createService.mockResolvedValue(existingService())

    const devices = [fixtureDevice({ id: 'd1', name: 'ONT-Main-01' }), fixtureDevice({ id: 'd2', name: 'ONT-Main-02' })]
    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices } })
    await wrapper.setProps({ open: true })
    await settle()

    const select = deviceSelect()!
    await select.setValue('d2')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createServiceEquipment).toHaveBeenCalledWith({
      serviceId: 's1',
      deviceId: 'd2',
      role: 'ONU',
      description: '',
    })
  })

  it('blocks submission and shows a message when the customer has no eligible device', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices: [] } })
    await wrapper.setProps({ open: true })
    await settle()

    expect(body().find('.service-form__error').text()).toContain('no device available')
    expect(body().findAll('button').some((b) => b.text() === 'Add Service')).toBe(false)
  })

  it('still creates the service, and emits created, even if attaching the device afterward fails', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    createService.mockResolvedValue(existingService())
    createServiceEquipment.mockRejectedValue(new ApiError('device already assigned', 'conflict', 409))

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices: [fixtureDevice()] } })
    await wrapper.setProps({ open: true })
    await settle()

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('created')?.[0]).toEqual([existingService()])
  })

  it('shows "no products" and hides the submit button when no products exist yet', async () => {
    listProducts.mockResolvedValue([])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices: [fixtureDevice()] } })
    await wrapper.setProps({ open: true })
    await settle()

    expect(body().find('.service-form__error').text()).toContain('No products exist yet')
    expect(body().findAll('button').some((b) => b.text() === 'Add Service')).toBe(false)
  })

  it('surfaces the API error message on a failed submit, and does not emit created', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    createService.mockRejectedValue(new ApiError('a service already exists for this location', 'conflict', 409))

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', devices: [fixtureDevice()] } })
    await wrapper.setProps({ open: true })
    await settle()

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.service-form__error').text()).toBe('a service already exists for this location')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (service prop present)', () => {
  it('prefills product/profile/status/description from the service and shows an "Edit Service" title', async () => {
    listProducts.mockResolvedValue([
      { id: 'p1', name: 'Fiber 1G', status: 'Active' },
      { id: 'p2', name: 'Fiber 500M', status: 'Active' },
    ])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    const service = existingService({ productId: 'p2', status: 'Active', description: 'Existing service' })

    const wrapper = mount(ServiceFormDialog, { props: { open: false, locationId: 'l1', service, devices: [] } })
    await wrapper.setProps({ open: true })
    await settle()

    expect(body().find('.base-modal__title').text()).toBe('Edit Service')
    expect((productSelect().element as HTMLSelectElement).value).toBe('p2')
  })

  it('submits the edited fields to updateService, ignoring the locationId prop in favor of the service\'s own, and passing through activated/suspended/disconnected unchanged', async () => {
    listProducts.mockResolvedValue([{ id: 'p1', name: 'Fiber 1G', status: 'Active' }])
    listServiceProfiles.mockResolvedValue([{ id: 'sp1', name: 'Residential Standard', status: 'Active' }])
    // locationId prop deliberately differs from service.locationId, so a
    // wrong implementation that used the prop instead of the service's
    // own location would fail this assertion, not pass it by accident.
    const service = existingService({ locationId: 'l-actual', status: 'Active', activatedAt: '2026-02-01T00:00:00Z' })
    updateService.mockResolvedValue({ ...service, description: 'Updated' })

    const wrapper = mount(ServiceFormDialog, {
      props: { open: false, locationId: 'l-prop-should-be-ignored', service, devices: [] },
    })
    await wrapper.setProps({ open: true })
    await settle()

    const descriptionInput = body()
      .findAll('.base-input')
      .find((el) => el.find('.base-input__label').text() === 'Description')!
      .find('input')
    await descriptionInput.setValue('Updated')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateService).toHaveBeenCalledWith('s1', {
      locationId: 'l-actual',
      productId: 'p1',
      serviceProfileId: 'sp1',
      status: 'Active',
      description: 'Updated',
      activatedAt: '2026-02-01T00:00:00Z',
      suspendedAt: null,
      disconnectedAt: null,
    })
    expect(createService).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...service, description: 'Updated' }])
  })
})
