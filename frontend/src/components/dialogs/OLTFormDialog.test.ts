import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import OLTFormDialog from './OLTFormDialog.vue'
import type { OLT } from '@/types/olt'

/**
 * Dual-mode, mirrors LocationFormDialog.test.ts's shape: takes a required
 * accessNetworkId prop, ignored in edit mode in favor of the OLT's own.
 */
const { createOLT, updateOLT } = vi.hoisted(() => ({ createOLT: vi.fn(), updateOLT: vi.fn() }))

vi.mock('@/services/olts/oltRepository', () => ({ createOLT, updateOLT }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingOLT(overrides: Partial<OLT> = {}): OLT {
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
  updateOLT.mockReset()
})

describe('create mode (no olt prop)', () => {
  it('defaults vendor to Nokia', () => {
    mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })

    expect((selectByLabel('Vendor').element as HTMLSelectElement).value).toBe('Nokia')
  })

  it('passes the accessNetworkId prop through into createOLT alongside the form fields, and emits created', async () => {
    createOLT.mockResolvedValue(existingOLT({ name: 'OLT-Core-1' }))
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
    expect(updateOLT).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingOLT({ name: 'OLT-Core-1' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createOLT.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.olt-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (olt prop present)', () => {
  it('prefills every field from the OLT and shows an "Edit OLT" title', () => {
    mount(OLTFormDialog, {
      props: { open: true, accessNetworkId: 'an1', olt: existingOLT({ name: 'OLT-Core-1', vendor: 'Calix' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit OLT')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('OLT-Core-1')
    expect((selectByLabel('Vendor').element as HTMLSelectElement).value).toBe('Calix')
  })

  it('submits the edited fields to updateOLT, using the OLT\'s own accessNetworkId/connectionProfileId rather than the prop, and emits updated', async () => {
    // accessNetworkId prop deliberately differs from olt.accessNetworkId,
    // so a wrong implementation that used the prop instead of the OLT's
    // own access network would fail this assertion, not pass it by
    // accident.
    const olt = existingOLT({ accessNetworkId: 'an-actual', connectionProfileId: 'cp1' })
    updateOLT.mockResolvedValue({ ...olt, name: 'OLT-Core-1 Renamed' })
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an-prop-should-be-ignored', olt } })

    await inputByLabel('Name').setValue('OLT-Core-1 Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateOLT).toHaveBeenCalledWith('olt1', {
      name: 'OLT-Core-1 Renamed',
      vendor: olt.vendor,
      model: olt.model,
      managementIpAddress: olt.managementIpAddress,
      description: olt.description,
      accessNetworkId: 'an-actual',
      connectionProfileId: 'cp1',
    })
    expect(createOLT).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...olt, name: 'OLT-Core-1 Renamed' }])
  })
})

it('closing and reopening for a different OLT repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(OLTFormDialog, {
    props: { open: true, accessNetworkId: 'an1', olt: existingOLT({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, olt: existingOLT({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
