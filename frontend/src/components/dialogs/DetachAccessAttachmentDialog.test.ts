import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import DetachAccessAttachmentDialog from './DetachAccessAttachmentDialog.vue'
import type { AccessAttachment } from '@/types/accessAttachment'

/**
 * No dual mode -- unlike DeviceFormDialog.vue, this dialog only ever
 * detaches the `attachment` it is given (no create path). Verifies the
 * PUT passes through accessInterfaceId/serviceEquipmentId/installedAt
 * unchanged and sets only removedAt/removalReason, per
 * accessAttachmentRepository.ts's own doc comment on updateAccessAttachment.
 */
const { updateAccessAttachment } = vi.hoisted(() => ({ updateAccessAttachment: vi.fn() }))

vi.mock('@/services/accessAttachments/accessAttachmentRepository', () => ({ updateAccessAttachment }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function existingAttachment(overrides: Partial<AccessAttachment> = {}): AccessAttachment {
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

function inputByLabel(labelText: string) {
  const label = body()
    .findAll('.base-input')
    .find((el) => el.find('.base-input__label').text() === labelText)
  if (!label) throw new Error(`no BaseInput labeled "${labelText}"`)
  return label.find('input')
}

beforeEach(() => {
  updateAccessAttachment.mockReset()
})

it('submits removedAt/removalReason while passing through the rest of the attachment unchanged, and emits detached', async () => {
  const attachment = existingAttachment()
  updateAccessAttachment.mockResolvedValue({ ...attachment, removedAt: '2026-03-01T00:00:00Z', removalReason: 'ONT swapped' })
  const wrapper = mount(DetachAccessAttachmentDialog, { props: { open: true, attachment } })

  await inputByLabel('Reason').setValue('ONT swapped')
  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(updateAccessAttachment).toHaveBeenCalledWith('aa1', {
    accessInterfaceId: 'ai1',
    serviceEquipmentId: 'se1',
    installedAt: '2026-01-01T00:00:00Z',
    removedAt: expect.any(String),
    removalReason: 'ONT swapped',
  })
  expect(wrapper.emitted('detached')?.[0]).toEqual([{ ...attachment, removedAt: '2026-03-01T00:00:00Z', removalReason: 'ONT swapped' }])
})

it('surfaces the API error message instead of throwing, and does not emit detached', async () => {
  updateAccessAttachment.mockRejectedValue(new ApiError('boom', 'internal', 500))
  const wrapper = mount(DetachAccessAttachmentDialog, { props: { open: true, attachment: existingAttachment() } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.detach-form__error').text()).toBe('boom')
  expect(wrapper.emitted('detached')).toBeUndefined()
})
