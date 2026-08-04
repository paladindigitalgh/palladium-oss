/**
 * The shared entry shape ActivityList and TimelineEntries
 * (components/data-display/) render -- both components already took no
 * opinion on what generated their entries. Promoted out of
 * types/customer.ts once devices needed the identical shape for their
 * own Recent Activity/Timeline sections
 * (docs/02-DESIGN-PRINCIPLES.md principle 10, "Events as Operational
 * History": the same event shape applies regardless of which domain
 * object it describes).
 */
export interface ActivityEntry {
  id: string
  label: string
  timestamp: string
  description?: string
}
