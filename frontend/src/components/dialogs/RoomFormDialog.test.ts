import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import RoomFormDialog from './RoomFormDialog.vue'
import type { Room } from '@/types/room'

/**
 * Dual-mode, mirrors BuildingFormDialog.test.ts's shape: takes a
 * required buildingId prop, ignored in edit mode in favor of the
 * Room's own.
 */
const { createRoom, updateRoom } = vi.hoisted(() => ({ createRoom: vi.fn(), updateRoom: vi.fn() }))

vi.mock('@/services/rooms/roomRepository', () => ({ createRoom, updateRoom }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingRoom(overrides: Partial<Room> = {}): Room {
  return {
    id: 'r1',
    buildingId: 'b1',
    name: 'First Floor',
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
  createRoom.mockReset()
  updateRoom.mockReset()
})

describe('create mode (no room prop)', () => {
  it('starts with the default field values', () => {
    mount(RoomFormDialog, { props: { open: true, buildingId: 'b1' } })

    expect(body().find('.base-modal__title').text()).toBe('Add Room')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  })

  it('passes the buildingId prop through into createRoom alongside the form fields, and emits created', async () => {
    createRoom.mockResolvedValue(existingRoom({ name: 'First Floor' }))
    const wrapper = mount(RoomFormDialog, { props: { open: true, buildingId: 'b1' } })

    await inputByLabel('Name').setValue('First Floor')
    await inputByLabel('Description').setValue('Main floor')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createRoom).toHaveBeenCalledWith({ buildingId: 'b1', name: 'First Floor', description: 'Main floor' })
    expect(updateRoom).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingRoom({ name: 'First Floor' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createRoom.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(RoomFormDialog, { props: { open: true, buildingId: 'b1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.room-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (room prop present)', () => {
  it('prefills every field from the room and shows an "Edit Room" title', () => {
    mount(RoomFormDialog, {
      props: { open: true, buildingId: 'b1', room: existingRoom({ name: 'First Floor', description: 'Main floor' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Room')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First Floor')
    expect((inputByLabel('Description').element as HTMLInputElement).value).toBe('Main floor')
  })

  it("submits the edited fields to updateRoom, using the room's own buildingId rather than the prop, and emits updated", async () => {
    // buildingId prop deliberately differs from room.buildingId, so a
    // wrong implementation that used the prop instead of the room's own
    // building would fail this assertion, not pass it by accident.
    const room = existingRoom({ buildingId: 'b-actual' })
    updateRoom.mockResolvedValue({ ...room, name: 'First Floor Renamed' })
    const wrapper = mount(RoomFormDialog, { props: { open: true, buildingId: 'b-prop-should-be-ignored', room } })

    await inputByLabel('Name').setValue('First Floor Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateRoom).toHaveBeenCalledWith('r1', {
      name: 'First Floor Renamed',
      description: room.description,
      buildingId: 'b-actual',
    })
    expect(createRoom).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...room, name: 'First Floor Renamed' }])
  })
})

it('closing and reopening for a different room repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(RoomFormDialog, {
    props: { open: true, buildingId: 'b1', room: existingRoom({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, room: existingRoom({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
