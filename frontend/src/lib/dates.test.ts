import { describe, it, expect } from 'vitest'
import { formatDisplayDate } from './dates'

/**
 * Regression coverage for a real bug found while building the Dashboard:
 * formatDisplayDate used to hand-split its input assuming a bare
 * "YYYY-MM-DD" string, a format nothing in this app has sent since the
 * mock dataset generators were deleted -- every real domain type's
 * timestamp is a full RFC3339 string off Go's time.Time, and the
 * timezone offset's own "-" broke the split, silently producing
 * "Invalid Date" everywhere a Created/Updated fact was rendered.
 *
 * Test timestamps are chosen well clear of local midnight (rather than
 * right at a UTC day boundary) so the expected calendar date holds
 * regardless of which timezone actually runs this suite.
 */
describe('formatDisplayDate', () => {
  it('formats a full RFC3339 timestamp with a negative timezone offset', () => {
    expect(formatDisplayDate('2026-09-03T10:53:06.10142-06:00')).toBe('Sep 3, 2026')
  })

  it('formats a full RFC3339 timestamp with a Z (UTC) suffix', () => {
    expect(formatDisplayDate('2026-01-15T12:00:00Z')).toBe('Jan 15, 2026')
  })

  it('never produces "Invalid Date" for a real backend timestamp', () => {
    expect(formatDisplayDate('2026-09-03T10:53:06.10142-06:00')).not.toMatch(/Invalid/)
  })
})
