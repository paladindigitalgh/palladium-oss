import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { ApiError } from '@/services/api/httpClient'
import DeviceFormDialog from './DeviceFormDialog.vue'
import type { Device } from '@/types/device'
import type { Rack } from '@/types/rack'

/**
 * The reference test for a dialogs/*FormDialog.vue component: mount for
 * real (BaseModal/BaseInput/BaseSelect are not stubbed -- they are small
 * enough that stubbing them would just hide bugs at the seam between
 * this component and them), mock only the repository functions it calls.
 * Covers this component's one piece of real logic: dual create/edit mode
 * -- which fields prefill, which repository function is called, which
 * event is emitted.
 *
 * BaseModal renders through <Teleport to="body">, so the dialog's DOM
 * never appears under `wrapper`'s own element -- every query below goes
 * through a DOMWrapper over `document.body` instead, the standard
 * Vue Test Utils pattern for asserting on teleported content.
 */
const { createDevice, updateDevice } = vi.hoisted(() => ({ createDevice: vi.fn(), updateDevice: vi.fn() }))
const { listRacks } = vi.hoisted(() => ({ listRacks: vi.fn() }))

vi.mock('@/services/devices/deviceRepository', () => ({ createDevice, updateDevice }))
vi.mock('@/services/racks/rackRepository', () => ({ listRacks }))

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
    manufacturer: 'Nokia',
    model: 'G-010G',
    serialNumber: 'SN123',
    assetTag: 'AT-1',
    status: 'Installed',
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
  listRacks.mockReset()
  listRacks.mockResolvedValue([])
})

describe('create mode (no device prop)', () => {
  it('starts with blank fields and a "New Device" title', () => {
    mount(DeviceFormDialog, { props: { open: true } })

    expect(body().find('.base-modal__title').text()).toBe('New Device')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('InStock')
  })

  it('submits the form fields to createDevice and emits created', async () => {
    createDevice.mockResolvedValue(existingDevice({ id: 'new-1', name: 'New ONT' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })

    await inputByLabel('Name').setValue('New ONT')
    await inputByLabel('Manufacturer').setValue('Nokia')
    await inputByLabel('Model').setValue('G-010G')
    await inputByLabel('Serial Number').setValue('SN999')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createDevice).toHaveBeenCalledWith({
      name: 'New ONT',
      manufacturer: 'Nokia',
      model: 'G-010G',
      serialNumber: 'SN999',
      assetTag: '',
      status: 'InStock',
      description: '',
      rackId: null,
    })
    expect(updateDevice).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingDevice({ id: 'new-1', name: 'New ONT' })])
  })

  it('surfaces the API error message instead of throwing', async () => {
    createDevice.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.device-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (device prop present)', () => {
  it('prefills every field from the device and shows an "Edit Device" title', () => {
    mount(DeviceFormDialog, { props: { open: true, device: existingDevice() } })

    expect(body().find('.base-modal__title').text()).toBe('Edit Device')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('ONT-1')
    expect((inputByLabel('Serial Number').element as HTMLInputElement).value).toBe('SN123')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Installed')
  })

  it('submits the edited fields to updateDevice, preserving the existing rackId, and emits updated', async () => {
    const device = existingDevice({ rackId: 'rack-1' })
    updateDevice.mockResolvedValue({ ...device, name: 'Renamed ONT' })
    const wrapper = mount(DeviceFormDialog, { props: { open: true, device } })

    await inputByLabel('Name').setValue('Renamed ONT')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateDevice).toHaveBeenCalledWith('d1', {
      name: 'Renamed ONT',
      manufacturer: device.manufacturer,
      model: device.model,
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
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, device: existingDevice({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})

describe('Rack field', () => {
  it('fetches racks when the dialog opens and offers a "None" option first', async () => {
    listRacks.mockResolvedValue([existingRack({ id: 'rack-1', name: 'Rack A' }), existingRack({ id: 'rack-2', name: 'Rack B' })])
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })

    await wrapper.setProps({ open: true })
    await settle()

    const options = selectByLabel('Rack').findAll('option')
    expect(options.map((option) => option.text())).toEqual(['None', 'Rack A', 'Rack B'])
  })

  it('defaults to "None" in create mode and submits a null rackId', async () => {
    createDevice.mockResolvedValue(existingDevice({ id: 'new-1' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: true } })

    await inputByLabel('Name').setValue('New ONT')
    await inputByLabel('Manufacturer').setValue('Nokia')
    await inputByLabel('Model').setValue('G-010G')
    await inputByLabel('Serial Number').setValue('SN999')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect((createDevice.mock.calls[0][0] as { rackId: string | null }).rackId).toBeNull()
  })

  it('submits the chosen rack on create', async () => {
    listRacks.mockResolvedValue([existingRack({ id: 'rack-1', name: 'Rack A' })])
    createDevice.mockResolvedValue(existingDevice({ id: 'new-1', rackId: 'rack-1' }))
    const wrapper = mount(DeviceFormDialog, { props: { open: false } })
    await wrapper.setProps({ open: true })
    await settle()

    await inputByLabel('Name').setValue('New ONT')
    await inputByLabel('Manufacturer').setValue('Nokia')
    await inputByLabel('Model').setValue('G-010G')
    await inputByLabel('Serial Number').setValue('SN999')
    await selectByLabel('Rack').setValue('rack-1')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect((createDevice.mock.calls[0][0] as { rackId: string | null }).rackId).toBe('rack-1')
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
