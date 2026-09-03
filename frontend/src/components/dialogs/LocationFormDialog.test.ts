import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import LocationFormDialog from './LocationFormDialog.vue'
import type { Location } from '@/types/location'

/** Create-only, takes a required customerId prop. Mirrors CustomerFormDialog.test.ts's shape. */
const { createLocation } = vi.hoisted(() => ({ createLocation: vi.fn() }))

vi.mock('@/services/locations/locationRepository', () => ({ createLocation }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newLocation(overrides: Partial<Location> = {}): Location {
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
})

it('passes the customerId prop through into createLocation alongside the form fields, and emits created', async () => {
  createLocation.mockResolvedValue(newLocation({ name: 'Warehouse' }))
  const wrapper = mount(LocationFormDialog, { props: { open: true, customerId: 'c1' } })

  await inputByLabel('Name').setValue('Warehouse')
  await inputByLabel('City').setValue('Springfield')
  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createLocation).toHaveBeenCalledWith(
    expect.objectContaining({ customerId: 'c1', name: 'Warehouse', city: 'Springfield' }),
  )
  expect(wrapper.emitted('created')?.[0]).toEqual([newLocation({ name: 'Warehouse' })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createLocation.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
  const wrapper = mount(LocationFormDialog, { props: { open: true, customerId: 'c1' } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.location-form__error').text()).toBe('name is required')
  expect(wrapper.emitted('created')).toBeUndefined()
})
