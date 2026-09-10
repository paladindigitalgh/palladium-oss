import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { ApiError } from '@/services/api/httpClient'
import DeviceFormDialog from './DeviceFormDialog.vue'
import type { Device } from '@/types/device'
import type { Rack } from '@/types/rack'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'
import type { DeviceModel } from '@/types/deviceModel'

/**
 * The reference test for a dialogs/*FormDialog.vue component: mount for
 * real (BaseModal/BaseInput/BaseSelect are not stubbed -- they are small
 * enough that stubbing them would just hide bugs at the seam between
 * this component and them), mock only the repository functions it calls.
 * Covers this component's one piece of real logic: dual create/edit mode
 * -- which fields prefill, which repository function is called, which
 * event is emitted -- plus the Manufacturer/Model cascading picker.
 *
 * BaseModal renders through <Teleport to="body">, so the dialog's DOM
 * never appears under `wrapper`'s own element -- every query below goes
 * through a DOMWrapper over `document.body` instead, the standard
 * Vue Test Utils pattern for asserting on teleported content.
 */
const { createDevice, updateDevice, authorizeAndCreateDevice } = vi.hoisted(() => ({
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
  authorizeAndCreateDevice: vi.fn(),
}))
const { listRacks } = vi.hoisted(() => ({ listRacks: vi.fn() }))
const { listDeviceManufacturers } = vi.hoisted(() => ({ listDeviceManufacturers: vi.fn() }))
const { listDeviceModels } = vi.hoisted(() => ({ listDeviceModels: vi.fn() }))
const { getAggregatedBlacklist } = vi.hoisted(() => ({ getAggregatedBlacklist: vi.fn() }))

vi.mock('@/services/devices/deviceRepository', () => ({ createDevice, updateDevice, authorizeAndCreateDevice }))
vi.mock('@/services/racks/rackRepository', () => ({ listRacks }))
vi.mock('@/services/deviceManufacturers/deviceManufacturerRepository', () => ({ listDeviceManufacturers }))
vi.mock('@/services/deviceModels/deviceModelRepository', () => ({ listDeviceModels }))
vi.mock('@/services/diagnostics/diagnosticsRepository', () => ({ getAggregatedBlacklist }))

function body() {
  return new DOMWrapper(document.body)
}

// BaseModal teleports to document.body, outside the mounted wrapper's own
// element -- unmount does not happen automatically between tests, so
// without this a previous test's teleported DOM would still be sitting
// in document.body when the next test's body() queries run.
enableAutoUnmount(afterEach)

async function settle() {
  await nextTick()
  await nextTick()
  await nextTick()
}

function existingDevice(overrides: Partial<Device> = {}): Device {
  return {
    id: 'd1',
    name: 'ONT-1',
    description: 'Lobby ONT',
    rackId: null,
    deviceModelId: 'model-nokia-g010g',
    manufacturer: 'Nokia',
    model: 'G-010G',
    serialNumber: 'SN123',
    assetTag: 'AT-1',
    status: 'Active',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function existingRack(overrides: Partial<Rack> = {}): Rack {
  return {
    id: 'rack-1',
    roomId: 'r1',
    name: 'Rack A',
    description: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function existingManufacturer(overrides: Partial<DeviceManufacturer> = {}): DeviceManufacturer {
  return {
    id: 'mfr-nokia',
    name: 'Nokia',
    description: '',
    isDefault: false,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function existingModel(overrides: Partial<DeviceModel> = {}): DeviceModel {
  return {
    id: 'model-nokia-g010g',
    manufacturerId: 'mfr-nokia',
    name: 'G-010G',
    description: '',
    isDefault: false,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function inputByLabel(labelText: string) {
  const label = body()
    .findAll('.base-input')
    .find((el) => el.find('.base-input__label').text() === labelText)
  if (!label) throw new Error(`no BaseInput labeled "${labelText}"`)
  return label.find('input')
}

function selectByLabel(labelText: string) {
  const label = body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === labelText)
  if (!label) throw new Error(`no BaseSelect labeled "${labelText}"`)
  return label.find('select')
}

beforeEach(() => {
  createDevice.mockReset()
  updateDevice.mockReset()
  authorizeAndCreateDevice.mockReset()
  listRacks.mockReset()
  listRacks.mockResolvedValue([])
  listDeviceManufacturers.mockReset()
  listDeviceManufacturers.mockResolvedValue([existingManufacturer()])
  listDeviceModels.mockReset()
  listDeviceModels.mockResolvedValue([existingModel()])
  getAggregatedBlacklist.mockReset()
  getAggregatedBlacklist.mockResolvedValue({ onus: [], unreachableOlts: [] })
})

describe('create mode (no device prop)', () => {
  it('starts with blank fields and a "New Device" title', () => {
    mount(DeviceFormDialog, { props: { open: true } })

    expect(body().find('.base-modal__title').text()).toBe('New Device')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  })

  it('does not render Rack, Asset Tag, or Status -- New Device is scoped to CPE, not shelf inventory', () => {
    mount(DeviceFormDialog, { props: { open: true } })

    const selectLabels = body()
      .findAll('.base-select')
      .map((el) => el.find('.base-select__label').text())
    expect(selectLabels).not.toContain('Rack')
    expect(selectLabels).not.toContain('Status')
    const inputLabels = body()
      .findAll('.base-input')
      .map((el) => el.find('.base-input__label').text())
    expect(inputLabels).not.toContain('Asset Tag')
  })

  it('submits the form fields to createDevice, defaulting Status to Unused, and emits created', async () => {
    createDevice.mockResolvedValue(existingDevice({ id: 'new-1', name: 'New ONT', status: 'Unused' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await inputByLabel('Name').setValue('New ONT')
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia-g010g')
    await inputByLabel('Serial Number').setValue('SN999')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createDevice).toHaveBeenCalledWith({
      name: 'New ONT',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'SN999',
      assetTag: '',
      status: 'Unused',
      description: '',
      rackId: null,
    })
    expect(updateDevice).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingDevice({ id: 'new-1', name: 'New ONT', status: 'Unused' })])
  })

  it('surfaces the API error message instead of throwing', async () => {
    createDevice.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.device-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (device prop present)', () => {
  it('prefills every field from the device and shows an "Edit Device" title', async () => {
    mount(DeviceFormDialog, { props: { open: true, device: existingDevice() } })
    await settle()

    expect(body().find('.base-modal__title').text()).toBe('Edit Device')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('ONT-1')
    expect((inputByLabel('Serial Number').element as HTMLInputElement).value).toBe('SN123')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
    expect((selectByLabel('Manufacturer').element as HTMLSelectElement).value).toBe('mfr-nokia')
    expect((selectByLabel('Model').element as HTMLSelectElement).value).toBe('model-nokia-g010g')
  })

  it('renders Rack, Asset Tag, and Status -- unlike New Device, Edit Device keeps them as a correction path', async () => {
    mount(DeviceFormDialog, { props: { open: true, device: existingDevice() } })
    await settle()

    const selectLabels = body()
      .findAll('.base-select')
      .map((el) => el.find('.base-select__label').text())
    expect(selectLabels).toContain('Rack')
    expect(selectLabels).toContain('Status')
    const inputLabels = body()
      .findAll('.base-input')
      .map((el) => el.find('.base-input__label').text())
    expect(inputLabels).toContain('Asset Tag')
  })

  it('submits the edited fields to updateDevice, preserving the existing rackId and deviceModelId, and emits updated', async () => {
    const device = existingDevice({ rackId: 'rack-1' })
    updateDevice.mockResolvedValue({ ...device, name: 'Renamed ONT' })
    const wrapper = mount(DeviceFormDialog, { props: { open: true, device } })
    await settle()

    await inputByLabel('Name').setValue('Renamed ONT')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateDevice).toHaveBeenCalledWith('d1', {
      name: 'Renamed ONT',
      deviceModelId: device.deviceModelId,
      serialNumber: device.serialNumber,
      assetTag: device.assetTag,
      status: device.status,
      description: device.description,
      rackId: 'rack-1',
    })
    expect(createDevice).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...device, name: 'Renamed ONT' }])
  })
})

it('closing and reopening for a different device repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(DeviceFormDialog, { props: { open: true, device: existingDevice({ name: 'First' }) } })
  await settle()
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, device: existingDevice({ name: 'Second' }) })
  await settle()

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})

describe('Manufacturer / Model cascading picker', () => {
  it('fetches manufacturers and models when the dialog opens', async () => {
    listDeviceManufacturers.mockResolvedValue([
      existingManufacturer({ id: 'mfr-nokia', name: 'Nokia' }),
      existingManufacturer({ id: 'mfr-calix', name: 'Calix' }),
    ])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })

    await wrapper.setProps({ open: true })
    await settle()

    const options = selectByLabel('Manufacturer').findAll('option')
    expect(options.map((option) => option.text())).toEqual(['Nokia', 'Calix'])
  })

  it('narrows the Model picker to the selected Manufacturer', async () => {
    listDeviceManufacturers.mockResolvedValue([
      existingManufacturer({ id: 'mfr-nokia', name: 'Nokia' }),
      existingManufacturer({ id: 'mfr-calix', name: 'Calix' }),
    ])
    listDeviceModels.mockResolvedValue([
      existingModel({ id: 'model-nokia', manufacturerId: 'mfr-nokia', name: 'G-010G' }),
      existingModel({ id: 'model-calix', manufacturerId: 'mfr-calix', name: '716GE' }),
    ])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })
    await wrapper.setProps({ open: true })
    await settle()

    await selectByLabel('Manufacturer').setValue('mfr-calix')
    await nextTick()

    const options = selectByLabel('Model').findAll('option')
    expect(options.map((option) => option.text())).toEqual(['716GE'])
  })

  it('clears the selected Model when switching to a Manufacturer it does not belong to', async () => {
    listDeviceManufacturers.mockResolvedValue([
      existingManufacturer({ id: 'mfr-nokia', name: 'Nokia' }),
      existingManufacturer({ id: 'mfr-calix', name: 'Calix' }),
    ])
    listDeviceModels.mockResolvedValue([
      existingModel({ id: 'model-nokia', manufacturerId: 'mfr-nokia', name: 'G-010G' }),
      existingModel({ id: 'model-calix', manufacturerId: 'mfr-calix', name: '716GE' }),
    ])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })
    await wrapper.setProps({ open: true })
    await settle()

    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia')
    await nextTick()

    await selectByLabel('Manufacturer').setValue('mfr-calix')
    await nextTick()

    expect((selectByLabel('Model').element as HTMLSelectElement).value).toBe('')
  })

  it('pre-selects the default Manufacturer and its default Model in create mode, still overridable', async () => {
    listDeviceManufacturers.mockResolvedValue([
      existingManufacturer({ id: 'mfr-nokia', name: 'Nokia' }),
      existingManufacturer({ id: 'mfr-calix', name: 'Calix', isDefault: true }),
    ])
    listDeviceModels.mockResolvedValue([
      existingModel({ id: 'model-nokia', manufacturerId: 'mfr-nokia', name: 'G-010G' }),
      existingModel({ id: 'model-calix-a', manufacturerId: 'mfr-calix', name: '716GE' }),
      existingModel({ id: 'model-calix-b', manufacturerId: 'mfr-calix', name: '844E', isDefault: true }),
    ])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })
    await wrapper.setProps({ open: true })
    await settle()

    expect((selectByLabel('Manufacturer').element as HTMLSelectElement).value).toBe('mfr-calix')
    expect((selectByLabel('Model').element as HTMLSelectElement).value).toBe('model-calix-b')

    // Still fully overridable -- picking a different Manufacturer clears
    // the auto-filled Model rather than leaving a mismatched selection.
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await nextTick()
    expect((selectByLabel('Model').element as HTMLSelectElement).value).toBe('')
  })

  it('leaves the pickers blank when no Manufacturer is marked default', async () => {
    listDeviceManufacturers.mockResolvedValue([existingManufacturer({ id: 'mfr-nokia', name: 'Nokia' })])
    listDeviceModels.mockResolvedValue([existingModel({ id: 'model-nokia', manufacturerId: 'mfr-nokia' })])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })
    await wrapper.setProps({ open: true })
    await settle()

    expect((selectByLabel('Manufacturer').element as HTMLSelectElement).value).toBe('')
    expect((selectByLabel('Model').element as HTMLSelectElement).value).toBe('')
  })

  it('disables submit until a Model is chosen', async () => {
    mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    const submitButton = body().findAll('button[type="submit"]')[0]
    expect(submitButton.attributes('disabled')).toBeDefined()

    await inputByLabel('Name').setValue('New ONT')
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia-g010g')
    await inputByLabel('Serial Number').setValue('SN999')
    await nextTick()

    expect(submitButton.attributes('disabled')).toBeUndefined()
  })
})

describe('Discovered ONU picker (create mode)', () => {
  it('does not render when the blacklist scan returns nothing', async () => {
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    expect(
      body()
        .findAll('.base-select')
        .some((el) => el.find('.base-select__label').text() === 'Discovered ONU'),
    ).toBe(false)
    expect(wrapper.exists()).toBe(true)
  })

  it('renders an option per blacklisted ONU plus "Enter manually" once the scan resolves', async () => {
    getAggregatedBlacklist.mockResolvedValue({
      onus: [
        { oltId: 'olt1', oltName: 'OLT-A', interface: 'xgs/6', serialNumber: 'ISKT001', registrationId: '', cause: 'unregistered' },
      ],
      unreachableOlts: [],
    })
    mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    const options = selectByLabel('Discovered ONU').findAll('option')
    expect(options.map((option) => option.text())).toEqual(['Enter manually', 'ISKT001 — OLT-A (xgs/6)'])
  })

  it('never renders in edit mode even when the blacklist has entries', async () => {
    getAggregatedBlacklist.mockResolvedValue({
      onus: [
        { oltId: 'olt1', oltName: 'OLT-A', interface: 'xgs/6', serialNumber: 'ISKT001', registrationId: '', cause: 'unregistered' },
      ],
      unreachableOlts: [],
    })
    mount(DeviceFormDialog, { props: { open: true, device: existingDevice() } })
    await settle()

    expect(getAggregatedBlacklist).not.toHaveBeenCalled()
    expect(
      body()
        .findAll('.base-select')
        .some((el) => el.find('.base-select__label').text() === 'Discovered ONU'),
    ).toBe(false)
  })

  it('a failed blacklist scan leaves the picker hidden without blocking the form', async () => {
    getAggregatedBlacklist.mockRejectedValue(new Error('every OLT unreachable'))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    expect(
      body()
        .findAll('.base-select')
        .some((el) => el.find('.base-select__label').text() === 'Discovered ONU'),
    ).toBe(false)
    expect(wrapper.exists()).toBe(true)
  })

  it('selecting a blacklisted ONU locks Serial Number to it', async () => {
    getAggregatedBlacklist.mockResolvedValue({
      onus: [
        { oltId: 'olt1', oltName: 'OLT-A', interface: 'xgs/6', serialNumber: 'ISKT001', registrationId: '', cause: 'unregistered' },
      ],
      unreachableOlts: [],
    })
    mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await selectByLabel('Discovered ONU').setValue('ISKT001')
    await nextTick()

    const serialInput = inputByLabel('Serial Number')
    expect((serialInput.element as HTMLInputElement).value).toBe('ISKT001')
    expect((serialInput.element as HTMLInputElement).disabled).toBe(true)
  })

  it('submits through authorizeAndCreateDevice with the ONU\'s oltId/interface when a Discovered ONU is selected, defaulting Status to Unused', async () => {
    getAggregatedBlacklist.mockResolvedValue({
      onus: [
        { oltId: 'olt1', oltName: 'OLT-A', interface: 'xgs/6', serialNumber: 'ISKT001', registrationId: '', cause: 'unregistered' },
      ],
      unreachableOlts: [],
    })
    authorizeAndCreateDevice.mockResolvedValue(existingDevice({ id: 'new-2', serialNumber: 'ISKT001', status: 'Unused' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await inputByLabel('Name').setValue('Discovered ONT')
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia-g010g')
    await selectByLabel('Discovered ONU').setValue('ISKT001')
    await nextTick()
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(authorizeAndCreateDevice).toHaveBeenCalledWith('olt1', 'xgs/6', {
      name: 'Discovered ONT',
      deviceModelId: 'model-nokia-g010g',
      serialNumber: 'ISKT001',
      assetTag: '',
      status: 'Unused',
      description: '',
      rackId: null,
    })
    expect(createDevice).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingDevice({ id: 'new-2', serialNumber: 'ISKT001', status: 'Unused' })])
  })

  it('switching back to "Enter manually" unlocks Serial Number and submits through createDevice', async () => {
    getAggregatedBlacklist.mockResolvedValue({
      onus: [
        { oltId: 'olt1', oltName: 'OLT-A', interface: 'xgs/6', serialNumber: 'ISKT001', registrationId: '', cause: 'unregistered' },
      ],
      unreachableOlts: [],
    })
    createDevice.mockResolvedValue(existingDevice({ id: 'new-3', serialNumber: 'SN999' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await selectByLabel('Discovered ONU').setValue('ISKT001')
    await nextTick()
    await selectByLabel('Discovered ONU').setValue('')
    await nextTick()

    expect((inputByLabel('Serial Number').element as HTMLInputElement).disabled).toBe(false)

    await inputByLabel('Name').setValue('Manual ONT')
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia-g010g')
    await inputByLabel('Serial Number').setValue('SN999')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createDevice).toHaveBeenCalledWith(
      expect.objectContaining({ serialNumber: 'SN999' }),
    )
    expect(authorizeAndCreateDevice).not.toHaveBeenCalled()
  })
})

describe('Rack field (edit mode only -- see field-visibility tests above)', () => {
  it('fetches racks when the dialog opens and offers a "None" option first', async () => {
    listRacks.mockResolvedValue([existingRack({ id: 'rack-1', name: 'Rack A' }), existingRack({ id: 'rack-2', name: 'Rack B' })])
    const wrapper = mount(DeviceFormDialog, { props: { open: false, device: existingDevice() } })

    await wrapper.setProps({ open: true })
    await settle()

    const options = selectByLabel('Rack').findAll('option')
    expect(options.map((option) => option.text())).toEqual(['None', 'Rack A', 'Rack B'])
  })

  it('always submits a null rackId on create, since Rack is not offered there', async () => {
    createDevice.mockResolvedValue(existingDevice({ id: 'new-1' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })
    await settle()

    await inputByLabel('Name').setValue('New ONT')
    await selectByLabel('Manufacturer').setValue('mfr-nokia')
    await selectByLabel('Model').setValue('model-nokia-g010g')
    await inputByLabel('Serial Number').setValue('SN999')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect((createDevice.mock.calls[0][0] as { rackId: string | null }).rackId).toBeNull()
  })

  it('prefills the Rack select from the device and submits "None" as a null rackId on edit', async () => {
    listRacks.mockResolvedValue([existingRack({ id: 'rack-1', name: 'Rack A' })])
    const device = existingDevice({ rackId: 'rack-1' })
    updateDevice.mockResolvedValue({ ...device, rackId: null })
    const wrapper = mount(DeviceFormDialog, { props: { open: false, device } })
    await wrapper.setProps({ open: true })
    await settle()

    expect((selectByLabel('Rack').element as HTMLSelectElement).value).toBe('rack-1')

    await selectByLabel('Rack').setValue('')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect((updateDevice.mock.calls[0][1] as { rackId: string | null }).rackId).toBeNull()
  })
})
