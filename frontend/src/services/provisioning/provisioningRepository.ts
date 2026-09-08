import { apiFetch } from '@/services/api/httpClient'

interface InterfaceResponseDto {
  interface: string
}

/**
 * Authorizes a physically-detected-but-unauthorized ONU on port
 * (internal/provisioning/kontron/httpapi.AuthorizationHandler) -- the
 * OLT assigns the next free ONU-ID index on port itself, so this returns
 * the actual interface the ONU landed on, not port verbatim. port is a
 * bare PON port (e.g. "xgs/6"), the same shape
 * BlacklistedONU.interface already is (see types/onuDiagnostics.ts).
 */
export async function authorizeONU(oltId: string, port: string, serialNumber: string): Promise<string> {
  const { interface: iface } = await apiFetch<InterfaceResponseDto>(`/provisioning/olts/${oltId}/authorize-onu`, {
    method: 'POST',
    body: { port, serial_number: serialNumber },
  })
  return iface
}

/**
 * Fully removes an ONU's authorization from its OLT
 * (internal/provisioning/kontron/httpapi.DeauthorizationHandler) --
 * resolved server-side from deviceId alone (which OLT, which interface),
 * unlike a diagnostic command which already needs an oltId/interface in
 * hand. On success this also marks the device's active
 * AccessAttachment/ServiceEquipment removed, so the caller should reload
 * whatever assignment data it is showing afterward. Returns the
 * interface that was deauthorized.
 */
export async function deauthorizeONU(deviceId: string): Promise<string> {
  const { interface: iface } = await apiFetch<InterfaceResponseDto>(
    `/provisioning/devices/${deviceId}/deauthorize-onu`,
    { method: 'POST' },
  )
  return iface
}
