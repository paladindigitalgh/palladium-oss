import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { ApiError } from '@/services/api/httpClient'
import AttachAccessAttachmentDialog from './AttachAccessAttachmentDialog.vue'
import type { AccessAttachment } from '@/types/accessAttachment'

/**
 * Mirrors ServiceFormDialog.test.ts's shape exactly: no `{ immediate: true }`
 * on its own `watch(() => props.open, ...)`, so every test here mounts
 * with `open: false` first, then `setProps({ open: true })`.
 */
const { createAccessAttachment } = vi.hoisted(() => ({ createAccessAttachment: vi.fn() }))
const { listServiceEquipment } = vi.hoisted(() => ({ listServiceEquipment: vi.fn() }))

vi.mock('@/services/accessAttachments/accessAttachmentRepository', () => ({ createAccessAttachment }))
vi.mock('@/services/serviceEquipment/serviceEquipmentRepository', () => ({ listServiceEquipment }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

async function settle() {
  await nextTick()
  await nextTick()
  await nextTick()
}

function newAttachment(overrides: Partial<AccessAttachment> = {}): AccessAttachment {
  return {
    id: 'aa1',
    accessInterfaceId: 'ai1',
    serviceEquipmentId: 'se1',
    installedAt: '2026-01-01T00:00:00Z',
    removedAt: null,
    removalReason: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  createAccessAttachment.mockReset()
  listServiceEquipment.mockReset()
})

it('preselects the first equipment once loaded, and submits it with the accessInterfaceId prop', async () => {
  listServiceEquipment.mockResolvedValue([{ id: 'se1', serviceId: 's1', deviceId: 'd1', role: 'ONU', description: '', installedAt: null, removedAt: null }])
  createAccessAttachment.mockResolvedValue(newAttachment())

  const wrapper = mount(AttachAccessAttachmentDialog, { props: { open: false, accessInterfaceId: 'ai1' } })
  await wrapper.setProps({ open: true })
  await settle()

  const equipmentSelect = body()
    .findAll('.base-select')
    .find((el) => el.find('.base-select__label').text() === 'Equipment')!
    .find('select')
  expect((equipmentSelect.element as HTMLSelectElement).value).toBe('se1')

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createAccessAttachment).toHaveBeenCalledWith({ accessInterfaceId: 'ai1', serviceEquipmentId: 'se1' })
  expect(wrapper.emitted('created')?.[0]).toEqual([newAttachment()])
})

it('shows "no service equipment" and hides the submit button when none exists yet', async () => {
  listServiceEquipment.mockResolvedValue([])

  const wrapper = mount(AttachAccessAttachmentDialog, { props: { open: false, accessInterfaceId: 'ai1' } })
  await wrapper.setProps({ open: true })
  await settle()

  expect(body().find('.attach-form__error').text()).toContain('No service equipment exists yet')
  expect(body().findAll('button').some((b) => b.text() === 'Attach Equipment')).toBe(false)
})

it('surfaces the API error message on a failed submit, and does not emit created', async () => {
  listServiceEquipment.mockResolvedValue([{ id: 'se1', serviceId: 's1', deviceId: 'd1', role: 'ONU', description: '', installedAt: null, removedAt: null }])
  createAccessAttachment.mockRejectedValue(new ApiError('this equipment is already attached', 'conflict', 409))

  const wrapper = mount(AttachAccessAttachmentDialog, { props: { open: false, accessInterfaceId: 'ai1' } })
  await wrapper.setProps({ open: true })
  await settle()

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.attach-form__error').text()).toBe('this equipment is already attached')
  expect(wrapper.emitted('created')).toBeUndefined()
})
