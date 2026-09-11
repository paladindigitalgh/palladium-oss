import { describe, it, expect } from 'vitest'
import { rowsToCsv } from './csv'

describe('rowsToCsv', () => {
  it('renders a header row from column labels', () => {
    const csv = rowsToCsv([{ key: 'name', label: 'Name' }], [])
    expect(csv).toBe('Name\n')
  })

  it('renders one line per row, reading each column by key', () => {
    const csv = rowsToCsv(
      [
        { key: 'name', label: 'Name' },
        { key: 'email', label: 'Email' },
      ],
      [{ name: 'Jane Doe', email: 'jane@example.com' }],
    )
    expect(csv).toBe('Name,Email\nJane Doe,jane@example.com\n')
  })

  it('defaults a missing field to an empty cell rather than throwing', () => {
    const csv = rowsToCsv([{ key: 'name', label: 'Name' }], [{}])
    expect(csv).toBe('Name\n\n')
  })

  it('quotes a field containing a comma', () => {
    const csv = rowsToCsv([{ key: 'v', label: 'V' }], [{ v: 'a,b' }])
    expect(csv).toBe('V\n"a,b"\n')
  })

  it('quotes a field containing a newline', () => {
    const csv = rowsToCsv([{ key: 'v', label: 'V' }], [{ v: 'a\nb' }])
    expect(csv).toBe('V\n"a\nb"\n')
  })

  it('quotes a field containing a double quote, doubling the embedded quote', () => {
    const csv = rowsToCsv([{ key: 'v', label: 'V' }], [{ v: 'a"b' }])
    expect(csv).toBe('V\n"a""b"\n')
  })

  it('leaves an ordinary field unquoted', () => {
    const csv = rowsToCsv([{ key: 'v', label: 'V' }], [{ v: 'plain value' }])
    expect(csv).toBe('V\nplain value\n')
  })
})
