import { it, expect, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import ConfirmationDialog from './ConfirmationDialog.vue'

/**
 * The simplest dialog -- no repository to mock at all, purely props in,
 * events out. Still mirrors DeviceFormDialog.test.ts's
 * DOMWrapper(document.body)/enableAutoUnmount pattern since it too goes
 * through BaseModal's Teleport.
 */
function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

const baseProps = { open: true, title: 'Delete Customer', description: 'Permanently delete Acme? This cannot be undone.' }

it('renders the title and description', () => {
  mount(ConfirmationDialog, { props: baseProps })

  expect(body().find('.base-modal__title').text()).toBe('Delete Customer')
  expect(body().find('.confirmation-dialog__description').text()).toBe('Permanently delete Acme? This cannot be undone.')
})

it('emits confirm when the confirm button is clicked', async () => {
  const wrapper = mount(ConfirmationDialog, { props: baseProps })

  await body().findAll('button').find((b) => b.text() === 'Confirm')!.trigger('click')

  expect(wrapper.emitted('confirm')).toHaveLength(1)
})

it('emits cancel when the cancel button is clicked', async () => {
  const wrapper = mount(ConfirmationDialog, { props: baseProps })

  await body().findAll('button').find((b) => b.text() === 'Cancel')!.trigger('click')

  expect(wrapper.emitted('cancel')).toHaveLength(1)
})

it('uses the primary button variant by default', () => {
  mount(ConfirmationDialog, { props: baseProps })

  const confirm = body().findAll('button').find((b) => b.text() === 'Confirm')!
  expect(confirm.classes()).toContain('base-button--primary')
  expect(confirm.classes()).not.toContain('base-button--destructive')
})

it('uses the destructive button variant when destructive is set', () => {
  mount(ConfirmationDialog, { props: { ...baseProps, destructive: true } })

  const confirm = body().findAll('button').find((b) => b.text() === 'Confirm')!
  expect(confirm.classes()).toContain('base-button--destructive')
})

it('disables both action buttons and shows "Working…" on the confirm button while pending', () => {
  // Scoped to .confirmation-dialog__actions, not every button on the
  // page: BaseModal's own header close ("x") button is unaffected by
  // `pending` -- only this dialog's own Cancel/Confirm are wired to it.
  mount(ConfirmationDialog, { props: { ...baseProps, pending: true } })

  const buttons = body().find('.confirmation-dialog__actions').findAll('button')
  expect(buttons.every((b) => (b.element as HTMLButtonElement).disabled)).toBe(true)
  expect(buttons.some((b) => b.text() === 'Working…')).toBe(true)
})

it('shows the error message when set, and shows nothing when it is null', () => {
  mount(ConfirmationDialog, { props: { ...baseProps, error: 'This customer still has locations attached.' } })
  expect(body().find('.confirmation-dialog__error').text()).toBe('This customer still has locations attached.')
})

it('renders no error element when error is not set', () => {
  mount(ConfirmationDialog, { props: baseProps })

  expect(body().find('.confirmation-dialog__error').exists()).toBe(false)
})
