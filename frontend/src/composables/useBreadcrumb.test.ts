import { describe, it, expect, beforeEach } from 'vitest'
import { setBreadcrumb, useBreadcrumb } from './useBreadcrumb'

/**
 * Module-scope singleton state (see useBreadcrumb.ts's own doc comment,
 * the same pattern useAuth.ts uses) -- reset via setBreadcrumb([]) in
 * beforeEach rather than vi.resetModules(), since nothing here is seeded
 * from an external source at module-load time the way useAuth.ts's
 * token is.
 */
beforeEach(() => {
  setBreadcrumb([])
})

it('starts empty', () => {
  const { items } = useBreadcrumb()

  expect(items.value).toEqual([])
})

it('setBreadcrumb updates every consumer of useBreadcrumb', () => {
  const { items } = useBreadcrumb()

  setBreadcrumb([{ label: 'Network', to: '/network' }, { label: 'OLT Details' }])

  expect(items.value).toEqual([{ label: 'Network', to: '/network' }, { label: 'OLT Details' }])
})

it('a later call replaces the trail entirely, not merges into it', () => {
  setBreadcrumb([{ label: 'Network', to: '/network' }, { label: 'OLT Details' }])

  setBreadcrumb([{ label: 'Devices', to: '/devices' }, { label: 'Details' }])

  const { items } = useBreadcrumb()
  expect(items.value).toEqual([{ label: 'Devices', to: '/devices' }, { label: 'Details' }])
})

describe('two components sharing the same singleton', () => {
  it('a second useBreadcrumb() call sees updates made through the first', () => {
    const first = useBreadcrumb()
    const second = useBreadcrumb()

    setBreadcrumb([{ label: 'Customers', to: '/customers' }, { label: 'Details' }])

    expect(second.items.value).toEqual(first.items.value)
    expect(second.items.value).toEqual([{ label: 'Customers', to: '/customers' }, { label: 'Details' }])
  })
})
