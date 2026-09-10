import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount, flushPromises } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import OLTFormDialog from './OLTFormDialog.vue'
import type { OLT } from '@/types/olt'
import type { OLTModel } from '@/types/oltModel'
import type { ConnectionProfile } from '@/types/connectionProfile'

/**
 * Dual-mode, mirrors LocationFormDialog.test.ts's shape: takes a required
 * accessNetworkId prop, ignored in edit mode in favor of the OLT's own.
 */
const { createOLT, updateOLT, listOLTModels, listConnectionProfiles } = vi.hoisted(() => ({
  createOLT: vi.fn(),
  updateOLT: vi.fn(),
  listOLTModels: vi.fn(),
  listConnectionProfiles: vi.fn(),
}))

vi.mock('@/services/olts/oltRepository', () => ({ createOLT, updateOLT }))
vi.mock('@/services/oltModels/oltModelRepository', () => ({ listOLTModels }))
vi.mock('@/services/connectionProfiles/connectionProfileRepository', () => ({ listConnectionProfiles }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingOLT(overrides: Partial<OLT> = {}): OLT {
  return {
    id: 'olt1',
    accessNetworkId: 'an1',
    name: 'OLT-Core-1',
    oltModelId: 'model-nokia',
    managementIpAddress: '10.0.0.1',
    connectionProfileId: null,
    description: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function existingOLTModel(overrides: Partial<OLTModel> = {}): OLTModel {
  return {
    id: 'model-nokia',
    vendor: 'Nokia',
    name: '7360 ISAM',
    ponPortCount: 16,
    description: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function existingConnectionProfile(overrides: Partial<ConnectionProfile> = {}): ConnectionProfile {
  return {
    id: 'cp1',
    name: 'Standard SSH',
    protocol: 'ssh',
    port: 22,
    authenticationId: null,
    timeout: '30s',
    hostKeyPolicy: 'Strict',
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
  listOLTModels.mockReset()
  listConnectionProfiles.mockReset()
  listOLTModels.mockResolvedValue([
    existingOLTModel({ id: 'model-nokia', vendor: 'Nokia', name: '7360 ISAM' }),
    existingOLTModel({ id: 'model-calix', vendor: 'Calix', name: 'E7-2' }),
  ])
  listConnectionProfiles.mockResolvedValue([existingConnectionProfile({ id: 'cp1', name: 'Standard SSH' })])
})

describe('create mode (no olt prop)', () => {
  it('defaults the OLT Model to the first one fetched, and the Connection Profile to None', async () => {
    mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })
    await flushPromises()

    expect((selectByLabel('OLT Model').element as HTMLSelectElement).value).toBe('model-nokia')
    expect((selectByLabel('Connection Profile').element as HTMLSelectElement).value).toBe('')
  })

  it('passes the accessNetworkId prop through into createOLT alongside the form fields, sending null for an unset Connection Profile, and emits created', async () => {
    createOLT.mockResolvedValue(existingOLT({ name: 'OLT-Core-1' }))
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })
    await flushPromises()

    await inputByLabel('Name').setValue('OLT-Core-1')
    await selectByLabel('OLT Model').setValue('model-calix')
    await inputByLabel('Management IP Address').setValue('10.0.0.5')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createOLT).toHaveBeenCalledWith({
      accessNetworkId: 'an1',
      name: 'OLT-Core-1',
      oltModelId: 'model-calix',
      managementIpAddress: '10.0.0.5',
      description: '',
      connectionProfileId: null,
    })
    expect(updateOLT).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingOLT({ name: 'OLT-Core-1' })])
  })

  it('sends the selected Connection Profile id when one is chosen', async () => {
    createOLT.mockResolvedValue(existingOLT({ name: 'OLT-Core-1', connectionProfileId: 'cp1' }))
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })
    await flushPromises()

    await inputByLabel('Name').setValue('OLT-Core-1')
    await selectByLabel('Connection Profile').setValue('cp1')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createOLT).toHaveBeenCalledWith(
      expect.objectContaining({ connectionProfileId: 'cp1' }),
    )
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createOLT.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1' } })
    await flushPromises()

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.olt-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (olt prop present)', () => {
  it('prefills every field from the OLT, including its Connection Profile, and shows an "Edit OLT" title', async () => {
    mount(OLTFormDialog, {
      props: {
        open: true,
        accessNetworkId: 'an1',
        olt: existingOLT({ name: 'OLT-Core-1', oltModelId: 'model-calix', connectionProfileId: 'cp1' }),
      },
    })
    await flushPromises()

    expect(body().find('.base-modal__title').text()).toBe('Edit OLT')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('OLT-Core-1')
    expect((selectByLabel('OLT Model').element as HTMLSelectElement).value).toBe('model-calix')
    expect((selectByLabel('Connection Profile').element as HTMLSelectElement).value).toBe('cp1')
  })

  it('defaults an unset Connection Profile to None rather than a blank/invalid selection', async () => {
    mount(OLTFormDialog, {
      props: { open: true, accessNetworkId: 'an1', olt: existingOLT({ connectionProfileId: null }) },
    })
    await flushPromises()

    expect((selectByLabel('Connection Profile').element as HTMLSelectElement).value).toBe('')
  })

  it('submits the edited fields to updateOLT, using the OLT\'s own accessNetworkId rather than the prop, and emits updated', async () => {
    // accessNetworkId prop deliberately differs from olt.accessNetworkId,
    // so a wrong implementation that used the prop instead of the OLT's
    // own access network would fail this assertion, not pass it by
    // accident.
    const olt = existingOLT({ accessNetworkId: 'an-actual', connectionProfileId: 'cp1' })
    updateOLT.mockResolvedValue({ ...olt, name: 'OLT-Core-1 Renamed' })
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an-prop-should-be-ignored', olt } })
    await flushPromises()

    await inputByLabel('Name').setValue('OLT-Core-1 Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateOLT).toHaveBeenCalledWith('olt1', {
      name: 'OLT-Core-1 Renamed',
      oltModelId: olt.oltModelId,
      managementIpAddress: olt.managementIpAddress,
      description: olt.description,
      accessNetworkId: 'an-actual',
      connectionProfileId: 'cp1',
    })
    expect(createOLT).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...olt, name: 'OLT-Core-1 Renamed' }])
  })

  it('clearing the Connection Profile back to None sends null to updateOLT', async () => {
    const olt = existingOLT({ connectionProfileId: 'cp1' })
    updateOLT.mockResolvedValue(olt)
    const wrapper = mount(OLTFormDialog, { props: { open: true, accessNetworkId: 'an1', olt } })
    await flushPromises()

    await selectByLabel('Connection Profile').setValue('')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateOLT).toHaveBeenCalledWith('olt1', expect.objectContaining({ connectionProfileId: null }))
  })
})

it('closing and reopening for a different OLT repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(OLTFormDialog, {
    props: { open: true, accessNetworkId: 'an1', olt: existingOLT({ name: 'First' }) },
  })
  await flushPromises()
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, olt: existingOLT({ name: 'Second' }) })
  await flushPromises()

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
