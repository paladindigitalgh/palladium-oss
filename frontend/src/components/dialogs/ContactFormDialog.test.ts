import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import ContactFormDialog from './ContactFormDialog.vue'
import type { Contact } from '@/types/contact'

/**
 * Dual-mode, built from the start (unlike Location/Device, which got
 * edit as a separate follow-up) -- mirrors LocationFormDialog.test.ts
 * exactly. `customerId` (the parent prop, needed for create) is ignored
 * in edit mode -- covered below by deliberately mismatching the prop and
 * the contact's own customerId, the same technique
 * LocationFormDialog.test.ts/ServiceFormDialog.test.ts use for their own
 * passthrough props.
 */
const { createContact, updateContact } = vi.hoisted(() => ({ createContact: vi.fn(), updateContact: vi.fn() }))

vi.mock('@/services/contacts/contactRepository', () => ({ createContact, updateContact }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingContact(overrides: Partial<Contact> = {}): Contact {
  return {
    id: 'ct1',
    customerId: 'c1',
    name: 'Jane Doe',
    role: 'Primary',
    email: 'jane@example.com',
    phone: '555-0100',
    status: 'Active',
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

function selectByLabel(labelText: string) {
  const label = body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === labelText)
  if (!label) throw new Error(`no BaseSelect labeled "${labelText}"`)
  return label.find('select')
}

beforeEach(() => {
  createContact.mockReset()
  updateContact.mockReset()
})

describe('create mode (no contact prop)', () => {
  it('starts with the default field values', () => {
    mount(ContactFormDialog, { props: { open: true, customerId: 'c1' } })

    expect(body().find('.base-modal__title').text()).toBe('Add Contact')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
    expect((selectByLabel('Role').element as HTMLSelectElement).value).toBe('Primary')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
  })

  it('passes the customerId prop through into createContact alongside the form fields, and emits created', async () => {
    createContact.mockResolvedValue(existingContact({ name: 'John Smith' }))
    const wrapper = mount(ContactFormDialog, { props: { open: true, customerId: 'c1' } })

    await inputByLabel('Name').setValue('John Smith')
    await selectByLabel('Role').setValue('Billing')
    await inputByLabel('Email').setValue('john@example.com')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createContact).toHaveBeenCalledWith({
      customerId: 'c1',
      name: 'John Smith',
      role: 'Billing',
      email: 'john@example.com',
      phone: '',
      status: 'Active',
      description: '',
    })
    expect(updateContact).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingContact({ name: 'John Smith' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createContact.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(ContactFormDialog, { props: { open: true, customerId: 'c1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.contact-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (contact prop present)', () => {
  it('prefills every field from the contact and shows an "Edit Contact" title', () => {
    mount(ContactFormDialog, {
      props: { open: true, customerId: 'c1', contact: existingContact({ name: 'Jane Doe', role: 'Technical', status: 'Inactive' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Contact')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Jane Doe')
    expect((selectByLabel('Role').element as HTMLSelectElement).value).toBe('Technical')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Inactive')
  })

  it('submits the edited fields to updateContact, using the contact\'s own customerId rather than the prop, and emits updated', async () => {
    // customerId prop deliberately differs from contact.customerId, so a
    // wrong implementation that used the prop instead of the contact's
    // own customer would fail this assertion, not pass it by accident.
    const contact = existingContact({ customerId: 'c-actual' })
    updateContact.mockResolvedValue({ ...contact, name: 'Jane Renamed' })
    const wrapper = mount(ContactFormDialog, { props: { open: true, customerId: 'c-prop-should-be-ignored', contact } })

    await inputByLabel('Name').setValue('Jane Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateContact).toHaveBeenCalledWith('ct1', {
      customerId: 'c-actual',
      name: 'Jane Renamed',
      role: contact.role,
      email: contact.email,
      phone: contact.phone,
      status: contact.status,
      description: contact.description,
    })
    expect(createContact).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...contact, name: 'Jane Renamed' }])
  })
})

it('closing and reopening for a different contact repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(ContactFormDialog, {
    props: { open: true, customerId: 'c1', contact: existingContact({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, contact: existingContact({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
