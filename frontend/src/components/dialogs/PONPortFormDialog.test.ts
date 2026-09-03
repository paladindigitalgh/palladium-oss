import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import PONPortFormDialog from './PONPortFormDialog.vue'
import type { PONPort } from '@/types/ponPort'

/**
 * Dual-mode, takes a required oltId prop ignored in edit mode. portNumber
 * is a text BaseInput (see PONPortFormDialog.vue -- BaseInput has no
 * numeric type) cast with Number() at submit, and populated via
 * String(ponPort.portNumber) when editing -- these tests check both
 * directions carry a real number, not a string.
 */
const { createPONPort, updatePONPort } = vi.hoisted(() => ({ createPONPort: vi.fn(), updatePONPort: vi.fn() }))

vi.mock('@/services/ponPorts/ponPortRepository', () => ({ createPONPort, updatePONPort }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingPONPort(overrides: Partial<PONPort> = {}): PONPort {
  return {
    id: 'pp1',
    oltId: 'olt1',
    portNumber: 1,
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

beforeEach(() => {
  createPONPort.mockReset()
  updatePONPort.mockReset()
})

describe('create mode (no ponPort prop)', () => {
  it('passes the oltId prop through into createPONPort, casting portNumber to a number, and emits created', async () => {
    createPONPort.mockResolvedValue(existingPONPort({ portNumber: 3 }))
    const wrapper = mount(PONPortFormDialog, { props: { open: true, oltId: 'olt1' } })

    await inputByLabel('Port Number').setValue('3')
    await inputByLabel('Description').setValue('Rack 2 PON port 3')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createPONPort).toHaveBeenCalledWith({ oltId: 'olt1', portNumber: 3, description: 'Rack 2 PON port 3' })
    expect(updatePONPort).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingPONPort({ portNumber: 3 })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createPONPort.mockRejectedValue(new ApiError('port_number must be greater than 0', 'invalid', 422))
    const wrapper = mount(PONPortFormDialog, { props: { open: true, oltId: 'olt1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.pon-port-form__error').text()).toBe('port_number must be greater than 0')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (ponPort prop present)', () => {
  it('prefills the port number (as a string) and description, and shows an "Edit PON Port" title', () => {
    mount(PONPortFormDialog, {
      props: { open: true, oltId: 'olt1', ponPort: existingPONPort({ portNumber: 5, description: 'Existing' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit PON Port')
    expect((inputByLabel('Port Number').element as HTMLInputElement).value).toBe('5')
    expect((inputByLabel('Description').element as HTMLInputElement).value).toBe('Existing')
  })

  it('submits the edited fields to updatePONPort, using the port\'s own oltId rather than the prop, and emits updated', async () => {
    // oltId prop deliberately differs from ponPort.oltId, so a wrong
    // implementation that used the prop instead of the port's own OLT
    // would fail this assertion, not pass it by accident.
    const ponPort = existingPONPort({ oltId: 'olt-actual', portNumber: 5 })
    updatePONPort.mockResolvedValue({ ...ponPort, portNumber: 9 })
    const wrapper = mount(PONPortFormDialog, { props: { open: true, oltId: 'olt-prop-should-be-ignored', ponPort } })

    await inputByLabel('Port Number').setValue('9')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updatePONPort).toHaveBeenCalledWith('pp1', { portNumber: 9, description: ponPort.description, oltId: 'olt-actual' })
    expect(createPONPort).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...ponPort, portNumber: 9 }])
  })
})

it('closing and reopening for a different PON port repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(PONPortFormDialog, {
    props: { open: true, oltId: 'olt1', ponPort: existingPONPort({ portNumber: 1 }) },
  })
  expect((inputByLabel('Port Number').element as HTMLInputElement).value).toBe('1')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, ponPort: existingPONPort({ portNumber: 2 }) })

  expect((inputByLabel('Port Number').element as HTMLInputElement).value).toBe('2')
})
