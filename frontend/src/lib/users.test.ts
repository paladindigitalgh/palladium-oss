import { describe, it, expect } from 'vitest'
import { formatDisplayName } from './users'

describe('formatDisplayName', () => {
  it('joins first and last name when both are set', () => {
    expect(formatDisplayName({ firstName: 'Jane', lastName: 'Doe', email: 'jane@example.com' })).toBe('Jane Doe')
  })

  it('uses just the first name when only it is set', () => {
    expect(formatDisplayName({ firstName: 'Jane', lastName: '', email: 'jane@example.com' })).toBe('Jane')
  })

  it('uses just the last name when only it is set', () => {
    expect(formatDisplayName({ firstName: '', lastName: 'Doe', email: 'jane@example.com' })).toBe('Doe')
  })

  it('falls back to email when neither is set', () => {
    expect(formatDisplayName({ firstName: '', lastName: '', email: 'jane@example.com' })).toBe('jane@example.com')
  })

  it('falls back to email when both are only whitespace', () => {
    expect(formatDisplayName({ firstName: '  ', lastName: ' ', email: 'jane@example.com' })).toBe('jane@example.com')
  })
})
