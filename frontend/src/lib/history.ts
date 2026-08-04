import type { ActivityEntry } from '@/types/activity'
import { formatAbsoluteDate, formatRelative } from './dates'

/**
 * Turns a raw list of {date, template} history entries into the
 * ActivityEntry[] pair ActivityList/TimelineEntries both render:
 * `timeline` is every entry, `activity` is just its newest slice --
 * Recent Activity is never an independently maintained list.
 *
 * Extracted once a third dataset generator
 * (services/services/serviceDataset.ts) needed the exact same
 * sort-and-format tail that services/customers/customerDataset.ts and
 * services/devices/deviceDataset.ts already had. Only this formatting
 * step is shared -- each generator keeps its own logic for *which*
 * events happened and when, since that genuinely differs per domain
 * (docs/11-COMPONENT-ARCHITECTURE.md, "Future Evolution": promote a
 * pattern once a second -- here, third -- place needs it).
 */
export interface HistoryEventTemplate {
  label: string
  description: string
}

export interface HistoryEntry {
  date: Date
  template: HistoryEventTemplate
}

export function finalizeHistory(
  idPrefix: string,
  entries: HistoryEntry[],
  activitySliceSize = 8,
): { timeline: ActivityEntry[]; activity: ActivityEntry[] } {
  const sorted = entries.slice().sort((a, b) => b.date.getTime() - a.date.getTime())

  const timeline: ActivityEntry[] = sorted.map((entry, index) => ({
    id: `${idPrefix}-EVT-${index + 1}`,
    label: entry.template.label,
    timestamp: formatAbsoluteDate(entry.date),
    description: entry.template.description,
  }))

  const activity: ActivityEntry[] = sorted.slice(0, Math.min(activitySliceSize, sorted.length)).map((entry, index) => ({
    id: `${idPrefix}-ACT-${index + 1}`,
    label: entry.template.label,
    timestamp: formatRelative(entry.date),
    description: entry.template.description,
  }))

  return { timeline, activity }
}
