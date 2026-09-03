import type { PONPort } from '@/types/ponPort'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real PONPort data source. Mirrors oltRepository.ts's shape exactly
 * -- GET /pon-ports has no server-side filtering, so "an OLT's PON
 * Ports" is resolved by fetching every PON Port once and filtering
 * client-side, and getPONPortById hits GET /pon-ports/:id directly since
 * PONPort has its own Detail page.
 */

interface PONPortDto {
  id: string
  olt_id: string
  port_number: number
  description: string
  created_at: string
  updated_at: string
}

function fromDto(dto: PONPortDto): PONPort {
  return {
    id: dto.id,
    oltId: dto.olt_id,
    portNumber: dto.port_number,
    description: dto.description,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export async function listPONPorts(): Promise<PONPort[]> {
  const { pon_ports: ponPorts } = await apiFetch<{ pon_ports: PONPortDto[] }>('/pon-ports/')
  return ponPorts.map(fromDto)
}

export async function listPONPortsByOLTId(oltId: string): Promise<PONPort[]> {
  const ponPorts = await listPONPorts()
  return ponPorts.filter((ponPort) => ponPort.oltId === oltId)
}

/** Fetches a single PONPort, returning null (not throwing) when it does not exist. */
export async function getPONPortById(id: string): Promise<PONPort | null> {
  try {
    const dto = await apiFetch<PONPortDto>(`/pon-ports/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreatePONPortInput {
  oltId: string
  portNumber: number
  description: string
}

export async function createPONPort(input: CreatePONPortInput): Promise<PONPort> {
  const dto = await apiFetch<PONPortDto>('/pon-ports/', {
    method: 'POST',
    body: { olt_id: input.oltId, port_number: input.portNumber, description: input.description },
  })
  return fromDto(dto)
}

export interface UpdatePONPortInput {
  portNumber: number
  description: string
  /**
   * Not user-editable (see PONPortFormDialog.vue -- no OLT picker exists
   * in this workspace). Callers pass the PON Port's current oltId through
   * unchanged: PUT replaces every mutable column (see
   * internal/ponport/postgres's Update), so omitting it here would
   * silently move the port to no OLT.
   */
  oltId: string
}

export async function updatePONPort(id: string, input: UpdatePONPortInput): Promise<PONPort> {
  const dto = await apiFetch<PONPortDto>(`/pon-ports/${id}`, {
    method: 'PUT',
    body: { olt_id: input.oltId, port_number: input.portNumber, description: input.description },
  })
  return fromDto(dto)
}

/**
 * Deletes the PONPort identified by id. pon_ports.id is referenced by
 * access_interfaces.pon_port_id ON DELETE RESTRICT, so this throws an
 * ApiError with kind "conflict" if the PON Port still has any Access
 * Interface.
 */
export async function deletePONPort(id: string): Promise<void> {
  await apiFetch<void>(`/pon-ports/${id}`, { method: 'DELETE' })
}
