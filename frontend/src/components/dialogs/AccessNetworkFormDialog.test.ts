import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import AccessNetworkFormDialog from './AccessNetworkFormDialog.vue'
import type { AccessNetwork } from '@/types/accessNetwork'

/** Dual-mode, mirrors CustomerFormDialog.test.ts exactly. */
const { createAccessNetwork, updateAccessNetwork } = vi.hoisted(() => ({
  createAccessNetwork: vi.fn(),
  updateAccessNetwork: vi.fn(),
}))

vi.mock('@/services/accessNetworks/accessNetworkRepository', () => ({ createAccessNetwork, updateAccessNetwork }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingAccessNetwork(overrides: Partial<AccessNetwork> = {}): AccessNetwork {
  return {
    id: 'an1',
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
  updateAccessNetwork.mockReset()
})

describe('create mode (no accessNetwork prop)', () => {
  it('starts with the default field values', () => {
    mount(AccessNetworkFormDialog, { props: { open: true } })

    expect(body().find('.base-modal__title').text()).toBe('New Access Network')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Active')
  })

  it('submits the form fields to createAccessNetwork and emits created', async () => {
    createAccessNetwork.mockResolvedValue(existingAccessNetwork({ id: 'new-1', name: 'Metro North' }))
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
    expect(updateAccessNetwork).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingAccessNetwork({ id: 'new-1', name: 'Metro North' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createAccessNetwork.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(AccessNetworkFormDialog, { props: { open: true } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.access-network-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (accessNetwork prop present)', () => {
  it('prefills every field from the access network and shows an "Edit Access Network" title', () => {
    mount(AccessNetworkFormDialog, {
      props: { open: true, accessNetwork: existingAccessNetwork({ name: 'Metro North', status: 'Inactive' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Access Network')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Metro North')
    expect((selectByLabel('Status').element as HTMLSelectElement).value).toBe('Inactive')
  })

  it('submits the edited fields to updateAccessNetwork and emits updated', async () => {
    const accessNetwork = existingAccessNetwork()
    updateAccessNetwork.mockResolvedValue({ ...accessNetwork, name: 'Metro North Renamed' })
    const wrapper = mount(AccessNetworkFormDialog, { props: { open: true, accessNetwork } })

    await inputByLabel('Name').setValue('Metro North Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateAccessNetwork).toHaveBeenCalledWith('an1', {
      name: 'Metro North Renamed',
      status: accessNetwork.status,
      description: accessNetwork.description,
    })
    expect(createAccessNetwork).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...accessNetwork, name: 'Metro North Renamed' }])
  })
})

it('closing and reopening for a different access network repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(AccessNetworkFormDialog, {
    props: { open: true, accessNetwork: existingAccessNetwork({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, accessNetwork: existingAccessNetwork({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
