import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import SiteFormDialog from './SiteFormDialog.vue'
import type { Site } from '@/types/site'

/** Dual-mode, mirrors AccessNetworkFormDialog.test.ts, minus the status field (Site has none). */
const { createSite, updateSite } = vi.hoisted(() => ({ createSite: vi.fn(), updateSite: vi.fn() }))

vi.mock('@/services/sites/siteRepository', () => ({ createSite, updateSite }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingSite(overrides: Partial<Site> = {}): Site {
  return {
    id: 's1',
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
  createSite.mockReset()
  updateSite.mockReset()
})

describe('create mode (no site prop)', () => {
  it('starts with the default field values', () => {
    mount(SiteFormDialog, { props: { open: true } })

    expect(body().find('.base-modal__title').text()).toBe('New Site')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('')
  })

  it('submits the form fields to createSite and emits created', async () => {
    createSite.mockResolvedValue(existingSite({ id: 'new-1', name: 'Main Office' }))
    const wrapper = mount(SiteFormDialog, { props: { open: true } })

    await inputByLabel('Name').setValue('Main Office')
    await inputByLabel('Description').setValue('HQ')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(createSite).toHaveBeenCalledWith({ name: 'Main Office', description: 'HQ' })
    expect(updateSite).not.toHaveBeenCalled()
    expect(wrapper.emitted('created')?.[0]).toEqual([existingSite({ id: 'new-1', name: 'Main Office' })])
  })

  it('surfaces the API error message instead of throwing, and does not emit created', async () => {
    createSite.mockRejectedValue(new ApiError('name is required', 'invalid', 422))
    const wrapper = mount(SiteFormDialog, { props: { open: true } })

    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(body().find('.site-form__error').text()).toBe('name is required')
    expect(wrapper.emitted('created')).toBeUndefined()
  })
})

describe('edit mode (site prop present)', () => {
  it('prefills every field from the site and shows an "Edit Site" title', () => {
    mount(SiteFormDialog, { props: { open: true, site: existingSite({ name: 'Main Office', description: 'HQ' }) } })

    expect(body().find('.base-modal__title').text()).toBe('Edit Site')
    expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Main Office')
    expect((inputByLabel('Description').element as HTMLInputElement).value).toBe('HQ')
  })

  it('submits the edited fields to updateSite and emits updated', async () => {
    const site = existingSite()
    updateSite.mockResolvedValue({ ...site, name: 'Renamed Office' })
    const wrapper = mount(SiteFormDialog, { props: { open: true, site } })

    await inputByLabel('Name').setValue('Renamed Office')
    await body().find('form').trigger('submit.prevent')
    await wrapper.vm.$nextTick()

    expect(updateSite).toHaveBeenCalledWith('s1', { name: 'Renamed Office', description: site.description })
    expect(createSite).not.toHaveBeenCalled()
    expect(wrapper.emitted('updated')?.[0]).toEqual([{ ...site, name: 'Renamed Office' }])
  })
})

it('closing and reopening for a different site repopulates the form instead of keeping stale values', async () => {
  const wrapper = mount(SiteFormDialog, { props: { open: true, site: existingSite({ name: 'First' }) } })
  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('First')

  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true, site: existingSite({ name: 'Second' }) })

  expect((inputByLabel('Name').element as HTMLInputElement).value).toBe('Second')
})
