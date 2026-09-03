import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import AccessNetworkFormDialog from './AccessNetworkFormDialog.vue'
import type { AccessNetwork } from '@/types/accessNetwork'

/** Create-only, mirrors CustomerFormDialog.test.ts exactly. */
const { createAccessNetwork } = vi.hoisted(() => ({ createAccessNetwork: vi.fn() }))

vi.mock('@/services/accessNetworks/accessNetworkRepository', () => ({ createAccessNetwork }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newAccessNetwork(overrides: Partial<AccessNetwork> = {}): AccessNetwork {
  return {
    id: 'new-1',
    name: 'Metro North',
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
  createAccessNetwork.mockReset()
})

it('starts with the default field values', () => {
  mount(AccessNetworkFormDialog, { props: { open: true } })

  expect(body().find('.base-modal__title').text()).toBe('New Access Network')
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
})

it('submits the form fields to createAccessNetwork and emits created', async () => {
  createAccessNetwork.mockResolvedValue(newAccessNetwork({ name: 'Metro North' }))
  const wrapper = mount(AccessNetworkFormDialog, { props: { open: true } })

  await inputByLabel('Name').setValue('Metro North')
  await selectByLabel('Status').setValue('Inactive')
  await inputByLabel('Description').setValue('North metro fiber ring')
  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createAccessNetwork).toHaveBeenCalledWith({
    name: 'Metro North',
    status: 'Inactive',
    description: 'North metro fiber ring',
  })
  expect(wrapper.emitted('created')?.[0]).toEqual([newAccessNetwork({ name: 'Metro North' })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createAccessNetwork.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
  const wrapper = mount(AccessNetworkFormDialog, { props: { open: true } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.access-network-form__error').text()).toBe('name is required')
  expect(wrapper.emitted('created')).toBeUndefined()
})
