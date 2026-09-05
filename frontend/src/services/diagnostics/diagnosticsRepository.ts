import type { CustomerEquipmentLocation } from '@/types/onuDiagnostics'
import { apiFetch } from '@/services/api/httpClient'

interface CustomerEquipmentLocationDto {
  service_equipment_id: string
  olt_id: string
  interface: string
}

function fromDto(dto: CustomerEquipmentLocationDto): CustomerEquipmentLocation {
  return { serviceEquipmentId: dto.service_equipment_id, oltId: dto.olt_id, interface: dto.interface }
}

/**
 * Resolves where a Customer's currently-attached equipment sits on the
 * access network (internal/accesstopology/httpapi) -- which OLT, and
 * which interface, per currently-active ServiceEquipment record. An
 * empty result is not an error: it means the Customer has nothing
 * attached to the access network yet.
 */
export async function listCustomerEquipmentLocations(customerId: string): Promise<CustomerEquipmentLocation[]> {
  const { locations } = await apiFetch<{ locations: CustomerEquipmentLocationDto[] }>(
    `/diagnostics/customers/${customerId}/equipment-locations`,
  )
  return locations.map(fromDto)
}

interface CommandOutputDto {
  output: string
}

/**
 * Runs one of internal/diagnostics/kontron/httpapi's per-interface
 * commands against oltId/iface and returns the device's raw output,
 * verbatim -- there is no parsing anywhere in this stack (see
 * internal/diagnostics/kontron's own "no command parsing" doc comment),
 * so none is added here either. Not exported: callers use the named
 * functions below, one per known-safe command, the same
 * "business intent over vendor commands" reasoning the backend's own
 * KontronService applies -- this repository does not expose a generic
 * "run any path" function.
 */
async function runOLTDiagnostic(oltId: string, path: string, iface: string): Promise<string> {
  const { output } = await apiFetch<CommandOutputDto>(`/diagnostics/olts/${oltId}/${path}`, {
    method: 'POST',
    body: { interface: iface },
  })
  return output
}

/**
 * Same as runOLTDiagnostic, for the two whole-OLT commands
 * (ONUSummary/ONUStatusSummary) that take no interface argument and so
 * send no request body at all -- see KontronHandler.runNoArgs.
 */
async function runOLTWideDiagnostic(oltId: string, path: string): Promise<string> {
  const { output } = await apiFetch<CommandOutputDto>(`/diagnostics/olts/${oltId}/${path}`, { method: 'POST' })
  return output
}

/** Runs "show onu interface all": one row per ONU on this OLT, across every PON port -- identity fields (serial, registration ID, IP, MAC, description). */
export function runONUSummary(oltId: string): Promise<string> {
  return runOLTWideDiagnostic(oltId, 'onu-summary')
}

/** Runs "show onu interface all status": the same one-row-per-ONU shape as runONUSummary, but with operational state, distance, and optical levels instead of identity fields. */
export function runONUStatusSummary(oltId: string): Promise<string> {
  return runOLTWideDiagnostic(oltId, 'onu-status-summary')
}

/** Runs "show run <interface>": the ONU's stored provisioning configuration. */
export function runONURunningConfig(oltId: string, iface: string): Promise<string> {
  return runOLTDiagnostic(oltId, 'onu-running-config', iface)
}

/** Runs "show onu interface <interface> status": operational state, distance, and optical levels. */
export function runONUStatus(oltId: string, iface: string): Promise<string> {
  return runOLTDiagnostic(oltId, 'onu-status', iface)
}

/** Runs "show onu interface <interface> eth all": the ONU's physical Ethernet port status. */
export function runONUEthernetPorts(oltId: string, iface: string): Promise<string> {
  return runOLTDiagnostic(oltId, 'onu-ethernet-ports', iface)
}

/** Runs "show dhcpsnooping interface <interface>": DHCP snooping table entries learned on the ONU's ports. */
export function runDHCPSnoopingEntries(oltId: string, iface: string): Promise<string> {
  return runOLTDiagnostic(oltId, 'dhcp-snooping-entries', iface)
}

/** Runs "show mac-addr-table interface <interface>": learned MAC address table entries for the ONU's ports. */
export function runMACAddressTableEntries(oltId: string, iface: string): Promise<string> {
  return runOLTDiagnostic(oltId, 'mac-address-table-entries', iface)
}
