/**
 * The shared read-only operator note shape. Promoted out of
 * types/customer.ts once devices needed the identical shape for their
 * own Notes section.
 */
export interface Note {
  id: string
  author: string
  timestamp: string
  body: string
}
