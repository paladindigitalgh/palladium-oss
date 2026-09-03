import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import CustomerFormDialog from './CustomerFormDialog.vue'
import type { Customer } from '@/types/customer'

/**
 * Create-only (unlike DeviceFormDialog.test.ts's dual mode -- this
 * dialog has no `customer` prop, only ever creates). Mirrors
 * DeviceFormDialog.test.ts's DOMWrapper(document.body)/enableAutoUnmount
 * pattern for the same reason: BaseModal renders through Teleport.
 */
const { createCustomer } = vi.hoisted(() => ({ createCustomer: vi.fn() }))

vi.mock('@/services/customers/customerRepository', () => ({ createCustomer }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newCustomer(overrides: Partial<Customer> = {}): Customer {
  return {
    id: 'new-1',
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
})

it('starts with the default field values', () => {
  mount(CustomerFormDialog, { props: { open: true } })

  expect(body().find('.base-modal__title').text()).toBe('New Customer')
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  expect((selectByLabel('Customer Type').element as HTMLSelectElement).value).toBe('Residential')
  expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
})

it('submits the form fields to createCustomer and emits created', async () => {
  createCustomer.mockResolvedValue(newCustomer({ name: 'Acme' }))
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
  expect(wrapper.emitted('created')?.[0]).toEqual([newCustomer({ name: 'Acme' })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createCustomer.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
  const wrapper = mount(CustomerFormDialog, { props: { open: true } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.customer-form__error').text()).toBe('name is required')
  expect(wrapper.emitted('created')).toBeUndefined()
})
