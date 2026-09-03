import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import BuildingFormDialog from './BuildingFormDialog.vue'
import type { Building } from '@/types/building'

/**
 * Dual-mode, mirrors OLTFormDialog.test.ts's shape: takes a required
 * siteId prop, ignored in edit mode in favor of the Building's own.
 */
const { createBuilding, updateBuilding } = vi.hoisted(() => ({ createBuilding: vi.fn(), updateBuilding: vi.fn() }))

vi.mock('@/services/buildings/buildingRepository', () => ({ createBuilding, updateBuilding }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingBuilding(overrides: Partial<Building> = {}): Building {
  return {
    id: 'b1',
    siteId: 's1',
    name: 'Main Office',
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
  createBuilding.mockReset()
  updateBuilding.mockReset()
})

describe('create mode (no building prop)', () => {
  it('starts with the default field values', () => {
    mount(BuildingFormDialog, { props: { open: true, siteId: 's1' } })

    expect(body().find('.base-modal__title').text()).toBe('Add Building')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  })

  it('passes the siteId prop through into createBuilding alongside the form fields, and emits created', async () => {
    createBuilding.mockResolvedValue(existingBuilding({ name: 'Main Office' }))
    const wrapper = mount(BuildingFormDialog, { props: { open: true, siteId: 's1' } })

    await inputByLabel('Name').setValue('Main Office')
    await inputByLabel('Description').setValue('HQ')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createBuilding).toHaveBeenCalledWith({ siteId: 's1', name: 'Main Office', description: 'HQ' })
    expect(updateBuilding).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingBuilding({ name: 'Main Office' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createBuilding.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(BuildingFormDialog, { props: { open: true, siteId: 's1' } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.building-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (building prop present)', () => {
  it('prefills every field from the building and shows an "Edit Building" title', () => {
    mount(BuildingFormDialog, {
      props: { open: true, siteId: 's1', building: existingBuilding({ name: 'Main Office', description: 'HQ' }) },
    })

    expect(body().find('.base-modal__title').text()).toBe('Edit Building')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Main Office')
    expect((inputByLabel('Description').element as HTMLInputElement).value).toBe('HQ')
  })

  it("submits the edited fields to updateBuilding, using the building's own siteId rather than the prop, and emits updated", async () => {
    // siteId prop deliberately differs from building.siteId, so a wrong
    // implementation that used the prop instead of the building's own
    // site would fail this assertion, not pass it by accident.
    const building = existingBuilding({ siteId: 's-actual' })
    updateBuilding.mockResolvedValue({ ...building, name: 'Main Office Renamed' })
    const wrapper = mount(BuildingFormDialog, { props: { open: true, siteId: 's-prop-should-be-ignored', building } })

    await inputByLabel('Name').setValue('Main Office Renamed')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateBuilding).toHaveBeenCalledWith('b1', {
      name: 'Main Office Renamed',
      description: building.description,
      siteId: 's-actual',
    })
    expect(createBuilding).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...building, name: 'Main Office Renamed' }])
  })
})

it('closing and reopening for a different building repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(BuildingFormDialog, {
    props: { open: true, siteId: 's1', building: existingBuilding({ name: 'First' }) },
  })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, building: existingBuilding({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
