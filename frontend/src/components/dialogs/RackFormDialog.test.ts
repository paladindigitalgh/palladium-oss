import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import RackFormDialog from './RackFormDialog.vue'
import type { Rack } from '@/types/rack'

/**
 * Dual-mode, mirrors RoomFormDialog.test.ts's shape: takes a required
 * roomId prop, ignored in edit mode in favor of the Rack's own.
 */
const { createRack, updateRack } = vi.hoisted(() => ({ createRack: vi.fn(), updateRack: vi.fn() }))

vi.mock('@/services/racks/rackRepository', () => ({ createRack, updateRack }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingRack(overrides: Partial<Rack> = {}): Rack {
  return {
    id: 'rk1',
    roomId: 'r1',
    name: 'Rack A',
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
  createRack.mockReset()
  updateRack.mockReset()
})

describe('create mode (no rack prop)', () => {
  it('starts with the default field values', () => {
    mount(RackFormDialog, { props: { open: true, roomId: 'r1' } })

    expect(body().find('.base-modal__title').text()).toBe('Add Rack')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  })

  it('passes the roomId prop through into createRack alongside the form fields, and emits created', async () => {
    createRack.mockResolvedValue(existingRack({ name: 'Rack A' }))
    const wrapper = mount(RackFormDialog, { props: { open: true, roomId: 'r1' } })

    await inputByLabel('Name').setValue('Rack A')
    await inputByLabel('Description').setValue('Row 1')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createRack).toHaveBeenCalledWith({ roomId: 'r1', name: 'Rack A', description: 'Row 1' })
    expect(updateRack).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingRack({ name: 'Rack A' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createRack.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(RackFormDialog, { props: { open: true, roomId: 'r1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.rack-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (rack prop present)', () => {
  it('prefills every field from the rack and shows an "Edit Rack" title', () => {
    mount(RackFormDialog, {
      props: { open: true, roomId: 'r1', rack: existingRack({ name: 'Rack A', description: 'Row 1' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Rack')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Rack A')
    expect((inputByLabel('Description').element as HTMLInputElement).value).toBe('Row 1')
  })

  it("submits the edited fields to updateRack, using the rack's own roomId rather than the prop, and emits updated", async () => {
    // roomId prop deliberately differs from rack.roomId, so a wrong
    // implementation that used the prop instead of the rack's own room
    // would fail this assertion, not pass it by accident.
    const rack = existingRack({ roomId: 'r-actual' })
    updateRack.mockResolvedValue({ ...rack, name: 'Rack A Renamed' })
    const wrapper = mount(RackFormDialog, { props: { open: true, roomId: 'r-prop-should-be-ignored', rack } })

    await inputByLabel('Name').setValue('Rack A Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateRack).toHaveBeenCalledWith('rk1', {
      name: 'Rack A Renamed',
      description: rack.description,
      roomId: 'r-actual',
    })
    expect(createRack).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...rack, name: 'Rack A Renamed' }])
  })

  it('submits a null roomId unchanged when editing a rack that was never assigned to a room', async () => {
    const rack = existingRack({ roomId: null })
    updateRack.mockResolvedValue(rack)
    const wrapper = mount(RackFormDialog, { props: { open: true, roomId: 'r-prop-should-be-ignored', rack } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateRack).toHaveBeenCalledWith('rk1', { name: rack.name, description: rack.description, roomId: null })
  })
})

it('closing and reopening for a different rack repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(RackFormDialog, {
    props: { open: true, roomId: 'r1', rack: existingRack({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, rack: existingRack({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
