/**
 * Shared date helpers for mock dataset generators
 * (services/customers/customerDataset.ts, services/devices/deviceDataset.ts).
 * Extracted once a second dataset needed the exact same "fixed NOW plus
 * relative/absolute formatting" logic (docs/11-COMPONENT-ARCHITECTURE.md,
 * "Future Evolution": promote a pattern once a second place needs it).
 *
 * NOW is fixed rather than read from Date.now(): every generated
 * timestamp (and its relative-time label) stays deterministic across
 * sessions instead of silently drifting each day the dev server happens
 * to run.
 */
export const NOW = new Date(2026, 7, 4)

export function addDays(date: Date, days: number): Date {
  return new Date(date.getTime() + days * 24 * 60 * 60 * 1000)
}

export function daysBetween(a: Date, b: Date): number {
  return Math.round((b.getTime() - a.getTime()) / (1000 * 60 * 60 * 24))
}

export function formatIsoDate(date: Date): string {
  return date.toISOString().slice(0, 10)
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

export function formatAbsoluteDate(date: Date): string {
  return `${MONTHS[date.getMonth()]} ${date.getDate()}, ${date.getFullYear()}`
}

export function formatRelative(date: Date): string {
  const minutes = Math.floor((NOW.getTime() - date.getTime()) / 60000)
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  const days = Math.floor(hours / 24)
  if (days < 2) return 'Yesterday'
  if (days < 30) return `${days} days ago`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} month${months === 1 ? '' : 's'} ago`
  return formatAbsoluteDate(date)
}
