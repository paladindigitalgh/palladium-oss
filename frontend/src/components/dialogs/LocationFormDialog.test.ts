import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import LocationFormDialog from './LocationFormDialog.vue'
import type { Location } from '@/types/location'

/**
 * Dual-mode, mirroring DeviceFormDialog.test.ts/CustomerFormDialog.test.ts:
 * create when no `location` prop, edit when present. `customerId` (the
 * parent prop, needed for create) is ignored in edit mode -- covered
 * below by deliberately mismatching the prop and the location's own
 * customerId, the same technique ServiceFormDialog.test.ts uses for its
 * locationId passthrough.
 */
const { createLocation, updateLocation } = vi.hoisted(() => ({ createLocation: vi.fn(), updateLocation: vi.fn() }))

vi.mock('@/services/locations/locationRepository', () => ({ createLocation, updateLocation }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingLocation(overrides: Partial<Location> = {}): Location {
  return {
    id: 'l1',
    customerId: 'c1',
    name: 'Main Office',
    type: 'Office',
    status: 'Active',
    address1: '123 Main St',
    address2: '',
    city: 'Springfield',
    state: 'IL',
    postalCode: '62704',
    country: 'US',
    description: '',
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

beforeEach(() => {
  createLocation.mockReset()
  updateLocation.mockReset()
})

describe('create mode (no location prop)', () => {
  it('passes the customerId prop through into createLocation alongside the form fields, and emits created', async () => {
    createLocation.mockResolvedValue(existingLocation({ name: 'Warehouse' }))
    const wrapper = mount(LocationFormDialog, { props: { open: true, customerId: 'c1' } })

    await inputByLabel('Name').setValue('Warehouse')
    await inputByLabel('City').setValue('Springfield')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createLocation).toHaveBeenCalledWith(
      expect.objectContaining({ customerId: 'c1', name: 'Warehouse', city: 'Springfield' }),
    )
    expect(updateLocation).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingLocation({ name: 'Warehouse' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createLocation.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(LocationFormDialog, { props: { open: true, customerId: 'c1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.location-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (location prop present)', () => {
  it('prefills every field from the location and shows an "Edit Location" title', () => {
    mount(LocationFormDialog, {
      props: { open: true, customerId: 'c1', location: existingLocation({ name: 'Main Office', city: 'Springfield' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Location')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Main Office')
    expect((inputByLabel('City').element as HTMLInputElement).value).toBe('Springfield')
  })

  it('submits the edited fields to updateLocation, using the location\'s own customerId rather than the prop, and emits updated', async () => {
    // customerId prop deliberately differs from location.customerId, so a
    // wrong implementation that used the prop instead of the location's
    // own customer would fail this assertion, not pass it by accident.
    const location = existingLocation({ customerId: 'c-actual' })
    updateLocation.mockResolvedValue({ ...location, name: 'Renamed Office' })
    const wrapper = mount(LocationFormDialog, { props: { open: true, customerId: 'c-prop-should-be-ignored', location } })

    await inputByLabel('Name').setValue('Renamed Office')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateLocation).toHaveBeenCalledWith('l1', {
      customerId: 'c-actual',
      name: 'Renamed Office',
      type: location.type,
      status: location.status,
      address1: location.address1,
      address2: location.address2,
      city: location.city,
      state: location.state,
      postalCode: location.postalCode,
      country: location.country,
      description: location.description,
    })
    expect(createLocation).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...location, name: 'Renamed Office' }])
  })
})

it('closing and reopening for a different location repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(LocationFormDialog, {
    props: { open: true, customerId: 'c1', location: existingLocation({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, location: existingLocation({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
