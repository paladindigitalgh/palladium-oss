import { it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import PONPortFormDialog from './PONPortFormDialog.vue'
import type { PONPort } from '@/types/ponPort'

/**
 * Create-only, takes a required oltId prop. portNumber is a text
 * BaseInput (see PONPortFormDialog.vue -- BaseInput has no numeric type)
 * cast with Number() at submit -- this test types a numeric string and
 * checks createPONPort receives a real number, not the string.
 */
const { createPONPort } = vi.hoisted(() => ({ createPONPort: vi.fn() }))

vi.mock('@/services/ponPorts/ponPortRepository', () => ({ createPONPort }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function newPONPort(overrides: Partial<PONPort> = {}): PONPort {
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
})

it('passes the oltId prop through into createPONPort, casting portNumber to a number, and emits created', async () => {
  createPONPort.mockResolvedValue(newPONPort({ portNumber: 3 }))
  const wrapper = mount(PONPortFormDialog, { props: { open: true, oltId: 'olt1' } })

  await inputByLabel('Port Number').setValue('3')
  await inputByLabel('Description').setValue('Rack 2 PON port 3')
  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(createPONPort).toHaveBeenCalledWith({ oltId: 'olt1', portNumber: 3, description: 'Rack 2 PON port 3' })
  expect(wrapper.emitted('created')?.[0]).toEqual([newPONPort({ portNumber: 3 })])
})

it('surfaces the API error message instead of throwing, and does not emit created', async () => {
  createPONPort.mockRejectedValue(new ApiError('port_number must be greater than 0', 'invalid', 422))
  const wrapper = mount(PONPortFormDialog, { props: { open: true, oltId: 'olt1' } })

  await body().find('form').trigger('submit.prevent')
  await wrapper.vm.$nextTick()

  expect(body().find('.pon-port-form__error').text()).toBe('port_number must be greater than 0')
  expect(wrapper.emitted('created')).toBeUndefined()
})
