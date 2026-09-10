/**
 * The Customer Device domain type (internal/customerdevice), matching
 * internal/customerdevice/httpapi/dto.go's customerDeviceResponse -- the
 * link between a Customer and the Device placed at their premises,
 * independent of whether any Service has been set up to use it yet (see
 * that package's own doc comment on why this is the one deliberate
 * exception to "never couple inventory directly to customers"). Mirrors
 * types/serviceEquipment.ts's shape one domain over: this record only
 * says a Device is at a Customer's premises and since when, never how it
 * is configured or what it delivers.
 */
export interface CustomerDevice {
  id: string
  customerId: string
  deviceId: string
  /** Which of customerId's own Locations this Device physically sits at, purely for an operator's own tracking -- null means "not recorded," a common, legitimate state. */
  locationId: string | null
  attachedAt: string | null
  detachedAt: string | null
}
