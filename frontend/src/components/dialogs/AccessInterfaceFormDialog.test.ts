import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import AccessInterfaceFormDialog from './AccessInterfaceFormDialog.vue'
import type { AccessInterface } from '@/types/accessInterface'

/**
 * Dual-mode, mirrors OLTFormDialog.test.ts's shape: takes a required
 * ponPortId prop, ignored in edit mode in favor of the access
 * interface's own.
 */
const { createAccessInterface, updateAccessInterface } = vi.hoisted(() => ({
  createAccessInterface: vi.fn(),
  updateAccessInterface: vi.fn(),
}))

vi.mock('@/services/accessInterfaces/accessInterfaceRepository', () => ({ createAccessInterface, updateAccessInterface }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingAccessInterface(overrides: Partial<AccessInterface> = {}): AccessInterface {
  return {
    id: 'ai1',
    ponPortId: 'pp1',
    technology: 'GPON',
    name: 'AI-1',
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
  createAccessInterface.mockReset()
  updateAccessInterface.mockReset()
})

describe('create mode (no accessInterface prop)', () => {
  it('defaults technology to GPON and status to Active', () => {
    mount(AccessInterfaceFormDialog, { props: { open: true, ponPortId: 'pp1' } })

    expect((selectByLabel('Technology').element as HTMLSelectElement).value).toBe('GPON')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
  })

  it('passes the ponPortId prop through into createAccessInterface alongside the form fields, and emits created', async () => {
    createAccessInterface.mockResolvedValue(existingAccessInterface({ name: 'AI-1' }))
    const wrapper = mount(AccessInterfaceFormDialog, { props: { open: true, ponPortId: 'pp1' } })

    await inputByLabel('Name').setValue('AI-1')
    await selectByLabel('Technology').setValue('XGSPON')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createAccessInterface).toHaveBeenCalledWith({
      ponPortId: 'pp1',
      technology: 'XGSPON',
      name: 'AI-1',
      status: 'Active',
      description: '',
    })
    expect(updateAccessInterface).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingAccessInterface({ name: 'AI-1' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createAccessInterface.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(AccessInterfaceFormDialog, { props: { open: true, ponPortId: 'pp1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.access-interface-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (accessInterface prop present)', () => {
  it('prefills every field from the access interface and shows an "Edit Access Interface" title', () => {
    mount(AccessInterfaceFormDialog, {
      props: { open: true, ponPortId: 'pp1', accessInterface: existingAccessInterface({ name: 'AI-1', technology: 'XGSPON' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Access Interface')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('AI-1')
    expect((selectByLabel('Technology').element as HTMLSelectElement).value).toBe('XGSPON')
  })

  it('submits the edited fields to updateAccessInterface, using the interface\'s own ponPortId rather than the prop, and emits updated', async () => {
    // ponPortId prop deliberately differs from accessInterface.ponPortId,
    // so a wrong implementation that used the prop instead of the
    // interface's own PON Port would fail this assertion, not pass it by
    // accident.
    const accessInterface = existingAccessInterface({ ponPortId: 'pp-actual' })
    updateAccessInterface.mockResolvedValue({ ...accessInterface, name: 'AI-1 Renamed' })
    const wrapper = mount(AccessInterfaceFormDialog, {
      props: { open: true, ponPortId: 'pp-prop-should-be-ignored', accessInterface },
    })

    await inputByLabel('Name').setValue('AI-1 Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateAccessInterface).toHaveBeenCalledWith('ai1', {
      technology: accessInterface.technology,
      name: 'AI-1 Renamed',
      status: accessInterface.status,
      description: accessInterface.description,
      ponPortId: 'pp-actual',
    })
    expect(createAccessInterface).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...accessInterface, name: 'AI-1 Renamed' }])
  })
})

it('closing and reopening for a different access interface repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(AccessInterfaceFormDialog, {
    props: { open: true, ponPortId: 'pp1', accessInterface: existingAccessInterface({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, accessInterface: existingAccessInterface({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
