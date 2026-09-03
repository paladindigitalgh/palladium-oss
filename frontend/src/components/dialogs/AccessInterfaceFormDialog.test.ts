import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import AccessInterfaceFormDialog from './AccessInterfaceFormDialog.vue'
import type { AccessInterface } from '@/types/accessInterface'

/** Create-only, takes a required ponPortId prop. Mirrors OLTFormDialog.test.ts's shape. */
const { createAccessInterface } = vi.hoisted(() => ({ createAccessInterface: vi.fn() }))

vi.mock('@/services/accessInterfaces/accessInterfaceRepository', () => ({ createAccessInterface }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newAccessInterface(overrides: Partial<AccessInterface> = {}): AccessInterface {
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
})

it('defaults technology to GPON and status to Active', () => {
  mount(AccessInterfaceFormDialog, { props: { open: true, ponPortId: 'pp1' } })

  expect((selectByLabel('Technology').element as HTMLSelectElement).value).toBe('GPON')
  expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
})

it('passes the ponPortId prop through into createAccessInterface alongside the form fields, and emits created', async () => {
  createAccessInterface.mockResolvedValue(newAccessInterface({ name: 'AI-1' }))
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
  expect(wrapper.emitted('created')?.[0]).toEqual([newAccessInterface({ name: 'AI-1' })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createAccessInterface.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
  const wrapper = mount(AccessInterfaceFormDialog, { props: { open: true, ponPortId: 'pp1' } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.access-interface-form__error').text()).toBe('name is required')
  expect(wrapper.emitted('created')).toBeUndefined()
})
