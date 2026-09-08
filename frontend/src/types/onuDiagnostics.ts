/**
 * Where one of a Customer's currently-attached pieces of equipment sits
 * on the access network (internal/accesstopology), matching
 * GET /diagnostics/customers/:customerId/equipment-locations's
 * "locations" entries. Not a persisted domain entity -- it is resolved
 * on demand from ServiceEquipment/AccessAttachment/AccessInterface/
 * PONPort, never stored as its own row.
 */
export interface CustomerEquipmentLocation {
  serviceEquipmentId: string
  oltId: string
  interface: string
}

/**
 * One physically-detected-but-unauthorized ONU, from
 * POST /diagnostics/onu-blacklist's "onus" entries
 * (internal/diagnostics/kontron/service.KontronService.AggregatedBlacklist).
 * `interface` here is a bare PON port (e.g. "xgs/6"), not a full
 * interface path with an ONU-ID index -- an unauthorized ONU has not
 * been assigned one yet, which is exactly the `port` value
 * POST /provisioning/olts/:oltId/authorize-onu expects.
 */
export interface BlacklistedONU {
  oltId: string
  oltName: string
  interface: string
  serialNumber: string
  registrationId: string
  cause: string
}

/** One Kontron OLT the blacklist scan could not reach, from the same response's "unreachable_olts" entries -- the scan is best-effort across every OLT, not all-or-nothing. */
export interface UnreachableOLT {
  oltId: string
  oltName: string
  reason: string
}
