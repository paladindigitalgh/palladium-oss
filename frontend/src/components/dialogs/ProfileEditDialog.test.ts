import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, DOMWrapper, enableAutoUnmount } from '@vue/test-utils'
import { ApiError } from '@/services/api/httpClient'
import { useCurrentUser } from '@/composables/useCurrentUser'
import ProfileEditDialog from './ProfileEditDialog.vue'
import type { User } from '@/types/user'

const { updateCurrentUserName, changeCurrentUserPassword } = vi.hoisted(() => ({
  updateCurrentUserName: vi.fn(),
  changeCurrentUserPassword: vi.fn(),
}))

vi.mock('@/services/users/userRepository', () => ({ updateCurrentUserName, changeCurrentUserPassword }))

function body() {
  return new DOMWrapper(document.body)
}

enableAutoUnmount(afterEach)

function currentUser(overrides: Partial<User> = {}): User {
  return {
    id: 'u1',
    email: 'jane@example.com',
    firstName: 'Jane',
    lastName: 'Doe',
    role: 'Operator',
    status: 'Active',
    createdAt: '2026-01-01T00:00:00Z',
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

function nameForm() {
  return body().findAll('form')[0]
}

function passwordForm() {
  return body().findAll('form')[1]
}

beforeEach(() => {
  updateCurrentUserName.mockReset()
  changeCurrentUserPassword.mockReset()
  useCurrentUser().clear()
})

it('prefills the name fields from the cached current user', () => {
  useCurrentUser().set(currentUser())
  mount(ProfileEditDialog, { props: { open: true } })

  expect((inputByLabel('First Name').element as HTMLInputElement).value).toBe('Jane')
  expect((inputByLabel('Last Name').element as HTMLInputElement).value).toBe('Doe')
})

it('leaves the name fields blank when there is no cached current user yet', () => {
  mount(ProfileEditDialog, { props: { open: true } })

  expect((inputByLabel('First Name').element as HTMLInputElement).value).toBe('')
  expect((inputByLabel('Last Name').element as HTMLInputElement).value).toBe('')
})

describe('saving a name', () => {
  it('submits the edited name and updates the shared current-user cache', async () => {
    useCurrentUser().set(currentUser())
    updateCurrentUserName.mockResolvedValue(currentUser({ firstName: 'Janet' }))
    mount(ProfileEditDialog, { props: { open: true } })

    await inputByLabel('First Name').setValue('Janet')
    await nameForm().trigger('submit.prevent')

    expect(updateCurrentUserName).toHaveBeenCalledWith({ firstName: 'Janet', lastName: 'Doe' })
    expect(useCurrentUser().user.value?.firstName).toBe('Janet')
    expect(body().find('.profile-edit__success').text()).toBe('Saved.')
  })

  it('surfaces the API error message instead of throwing', async () => {
    useCurrentUser().set(currentUser())
    updateCurrentUserName.mockRejectedValue(new ApiError('name too long', 'invalid', 422))
    mount(ProfileEditDialog, { props: { open: true } })

    await nameForm().trigger('submit.prevent')

    expect(body().find('.profile-edit__error').text()).toBe('name too long')
  })
})

describe('changing password', () => {
  it('rejects a mismatched confirmation without calling the API', async () => {
    useCurrentUser().set(currentUser())
    mount(ProfileEditDialog, { props: { open: true } })

    await inputByLabel('Current Password').setValue('old-pass')
    await inputByLabel('New Password').setValue('new-pass')
    await inputByLabel('Confirm New Password').setValue('does-not-match')
    await passwordForm().trigger('submit.prevent')

    expect(changeCurrentUserPassword).not.toHaveBeenCalled()
    expect(body().find('.profile-edit__error').text()).toMatch(/do not match/)
  })

  it('submits current/new password and clears the fields on success', async () => {
    useCurrentUser().set(currentUser())
    changeCurrentUserPassword.mockResolvedValue(currentUser())
    mount(ProfileEditDialog, { props: { open: true } })

    await inputByLabel('Current Password').setValue('old-pass')
    await inputByLabel('New Password').setValue('new-pass')
    await inputByLabel('Confirm New Password').setValue('new-pass')
    await passwordForm().trigger('submit.prevent')

    expect(changeCurrentUserPassword).toHaveBeenCalledWith({ currentPassword: 'old-pass', newPassword: 'new-pass' })
    expect((inputByLabel('Current Password').element as HTMLInputElement).value).toBe('')
    expect(body().find('.profile-edit__success').text()).toBe('Password changed.')
  })

  it('surfaces the API error message instead of throwing', async () => {
    useCurrentUser().set(currentUser())
    changeCurrentUserPassword.mockRejectedValue(new ApiError('current password is incorrect', 'invalid', 422))
    mount(ProfileEditDialog, { props: { open: true } })

    await inputByLabel('Current Password').setValue('wrong')
    await inputByLabel('New Password').setValue('new-pass')
    await inputByLabel('Confirm New Password').setValue('new-pass')
    await passwordForm().trigger('submit.prevent')

    expect(body().find('.profile-edit__error').text()).toBe('current password is incorrect')
  })
})

it('reopening resets both forms rather than keeping stale values', async () => {
  useCurrentUser().set(currentUser())
  const wrapper = mount(ProfileEditDialog, { props: { open: true } })

  await inputByLabel('Current Password').setValue('typed-but-not-submitted')
  await wrapper.setProps({ open: false })
  await wrapper.setProps({ open: true })

  expect((inputByLabel('Current Password').element as HTMLInputElement).value).toBe('')
})
