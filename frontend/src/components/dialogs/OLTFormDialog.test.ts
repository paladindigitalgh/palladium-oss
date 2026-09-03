import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import OLTFormDialog from './OLTFormDialog.vue'
import type { OLT } from '@/types/olt'

/** Create-only, takes a required accessNetworkId prop. Mirrors LocationFormDialog.test.ts's shape. */
const { createOLT } = vi.hoisted(() => ({ createOLT: vi.fn() }))

vi.mock('@/services/olts/oltRepository', () => ({ createOLT }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newOLT(overrides: Partial<OLT> = {}): OLT {
  return {
    id: 'olt1',
    accessNetworkId: 'an1',
    name: 'OLT-Core-1',
    vendor: 'Nokia',
    model: '7360 ISAM',
    managementIpAddress: '10.0.0.1',
    connectionProfileId: null,
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
  createOLT.mockReset()
})

it('defaults vendor to Nokia', () => {
  mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })

  expect((selectByLabel('Vendor').element as HTMLSelectElement).value).toBe('Nokia')
})

it('passes the accessNetworkId prop through into createOLT alongside the form fields, and emits created', async () => {
  createOLT.mockResolvedValue(newOLT({ name: 'OLT-Core-1' }))
  const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })

  await inputByLabel('Name').setValue('OLT-Core-1')
  await selectByLabel('Vendor').setValue('Calix')
  await inputByLabel('Model').setValue('E7-2')
  await inputByLabel('Management IP Address').setValue('10.0.0.5')
  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createOLT).toHaveBeenCalledWith({
    accessNetworkId: 'an1',
    name: 'OLT-Core-1',
    vendor: 'Calix',
    model: 'E7-2',
    managementIpAddress: '10.0.0.5',
    description: '',
  })
  expect(wrapper.emitted('created')?.[0]).toEqual([newOLT({ name: 'OLT-Core-1' })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createOLT.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
  const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.olt-form__error').text()).toBe('name is required')
  expect(wrapper.emitted('created')).toBeUndefined()
})
