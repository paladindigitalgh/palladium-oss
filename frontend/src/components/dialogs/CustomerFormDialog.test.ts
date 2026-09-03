import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import CustomerFormDialog from './CustomerFormDialog.vue'
import type { Customer } from '@/types/customer'

/**
 * Dual-mode, mirroring DeviceFormDialog.test.ts exactly: create when no
 * `customer` prop, edit when present. Same DOMWrapper(document.body)/
 * enableAutoUnmount pattern for the same reason -- BaseModal renders
 * through Teleport.
 */
const { createCustomer, updateCustomer } = vi.hoisted(() => ({ createCustomer: vi.fn(), updateCustomer: vi.fn() }))

vi.mock('@/services/customers/customerRepository', () => ({ createCustomer, updateCustomer }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingCustomer(overrides: Partial<Customer> = {}): Customer {
  return {
    id: 'c1',
    name: 'Acme',
    customerType: 'Residential',
    status: 'Active',
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
  createCustomer.mockReset()
  updateCustomer.mockReset()
})

describe('create mode (no customer prop)', () => {
  it('starts with the default field values', () => {
    mount(CustomerFormDialog, { props: { open: true } })

    expect(body().find('.base-modal__title').text()).toBe('New Customer')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
    expect((selectByLabel('Customer Type').element as HTMLSelectElement).value).toBe('Residential')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
  })

  it('submits the form fields to createCustomer and emits created', async () => {
    createCustomer.mockResolvedValue(existingCustomer({ id: 'new-1', name: 'Acme' }))
    const wrapper = mount(CustomerFormDialog, { props: { open: true } })

    await inputByLabel('Name').setValue('Acme')
    await selectByLabel('Customer Type').setValue('Business')
    await inputByLabel('Description').setValue('A widget maker')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createCustomer).toHaveBeenCalledWith({
      name: 'Acme',
      customerType: 'Business',
      status: 'Active',
      description: 'A widget maker',
    })
    expect(updateCustomer).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingCustomer({ id: 'new-1', name: 'Acme' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createCustomer.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(CustomerFormDialog, { props: { open: true } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.customer-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (customer prop present)', () => {
  it('prefills every field from the customer and shows an "Edit Customer" title', () => {
    mount(CustomerFormDialog, { props: { open: true, customer: existingCustomer({ name: 'Acme', status: 'Inactive' }) } })

    expect(body().find('.base-modal__title').text()).toBe('Edit Customer')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Acme')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Inactive')
  })

  it('submits the edited fields to updateCustomer and emits updated', async () => {
    const customer = existingCustomer()
    updateCustomer.mockResolvedValue({ ...customer, name: 'Acme Renamed' })
    const wrapper = mount(CustomerFormDialog, { props: { open: true, customer } })

    await inputByLabel('Name').setValue('Acme Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateCustomer).toHaveBeenCalledWith('c1', {
      name: 'Acme Renamed',
      customerType: customer.customerType,
      status: customer.status,
      description: customer.description,
    })
    expect(createCustomer).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...customer, name: 'Acme Renamed' }])
  })
})

it('closing and reopening for a different customer repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(CustomerFormDialog, { props: { open: true, customer: existingCustomer({ name: 'First' }) } })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, customer: existingCustomer({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
