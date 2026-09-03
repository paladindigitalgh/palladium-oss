import type { Device } from '@/types/device'
import { apiFetch, ApiError } from '@/services/api/httpClient'

/**
 * The real Device data source, replacing the mock dataset this file used
 * to read from. Mirrors customerRepository.ts's shape exactly.
 */

interface DeviceDto {
  id: string
  name: string
  description: string
  rack_id: string | null
  manufacturer: string
  model: string
  serial_number: string
  asset_tag: string
  status: Device['status']
  created_at: string
  updated_at: string
}

function fromDto(dto: DeviceDto): Device {
  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    rackId: dto.rack_id,
    manufacturer: dto.manufacturer,
    model: dto.model,
    serialNumber: dto.serial_number,
    assetTag: dto.asset_tag,
    status: dto.status,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

export interface DeviceListQuery {
  search?: string
  status?: Device['status'] | 'all'
  sortKey?: 'name' | 'status'
  sortDirection?: 'asc' | 'desc'
  page?: number
  pageSize?: number
}

export interface DeviceListResult {
  items: Device[]
  total: number
}

function matchesSearch(device: Device, term: string): boolean {
  const needle = term.trim().toLowerCase()
  if (!needle) return true
  return (
    device.name.toLowerCase().includes(needle) ||
    device.serialNumber.toLowerCase().includes(needle) ||
    device.manufacturer.toLowerCase().includes(needle) ||
    device.model.toLowerCase().includes(needle)
  )
}

function compareDevices(sortKey: NonNullable<DeviceListQuery['sortKey']>, direction: number) {
  return (a: Device, b: Device): number => {
    const comparison = sortKey === 'status' ? a.status.localeCompare(b.status) : a.name.localeCompare(b.name)
    return comparison * direction
  }
}

async function listAllDevices(): Promise<Device[]> {
  const { devices } = await apiFetch<{ devices: DeviceDto[] }>('/devices/')
  return devices.map(fromDto)
}

/** Returns every Device racked in the given Rack, for RackDetailView.vue's read-only Devices section. */
export async function listDevicesByRackId(rackId: string): Promise<Device[]> {
  const devices = await listAllDevices()
  return devices.filter((device) => device.rackId === rackId)
}

/** Fetches every Device and applies search/filter/sort/pagination client-side. */
export async function listDevices(query: DeviceListQuery = {}): Promise<DeviceListResult> {
  const { search = '', status = 'all', sortKey = 'name', sortDirection = 'asc', page = 1, pageSize = 15 } = query

  let results = (await listAllDevices()).filter((device) => matchesSearch(device, search))

  if (status !== 'all') results = results.filter((device) => device.status === status)

  results = results.slice().sort(compareDevices(sortKey, sortDirection === 'desc' ? -1 : 1))

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

/** Fetches a single Device, returning null (not throwing) when it does not exist. */
export async function getDeviceById(id: string): Promise<Device | null> {
  try {
    const dto = await apiFetch<DeviceDto>(`/devices/${id}`)
    return fromDto(dto)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateDeviceInput {
  name: string
  manufacturer: string
  model: string
  serialNumber: string
  assetTag: string
  status: Device['status']
  description: string
  rackId: string | null
}

export async function createDevice(input: CreateDeviceInput): Promise<Device> {
  const dto = await apiFetch<DeviceDto>('/devices/', {
    method: 'POST',
    body: {
      name: input.name,
      manufacturer: input.manufacturer,
      model: input.model,
      serial_number: input.serialNumber,
      asset_tag: input.assetTag,
      status: input.status,
      description: input.description,
      rack_id: input.rackId,
    },
  })
  return fromDto(dto)
}

export interface UpdateDeviceInput {
  name: string
  manufacturer: string
  model: string
  serialNumber: string
  assetTag: string
  status: Device['status']
  description: string
  /**
   * User-editable via DeviceFormDialog.vue's Rack picker. PUT replaces
   * every mutable column (see internal/inventory/postgres/device.go's
   * Update), so omitting it here would silently unrack an installed
   * device.
   */
  rackId: string | null
}

export async function updateDevice(id: string, input: UpdateDeviceInput): Promise<Device> {
  const dto = await apiFetch<DeviceDto>(`/devices/${id}`, {
    method: 'PUT',
    body: {
      name: input.name,
      manufacturer: input.manufacturer,
      model: input.model,
      serial_number: input.serialNumber,
      asset_tag: input.assetTag,
      status: input.status,
      description: input.description,
      rack_id: input.rackId,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the Device identified by id. Device is a leaf in the Inventory
 * hierarchy -- no other table's foreign key can ever block this delete
 * (see internal/inventory/postgres/device.go's Delete) -- so unlike
 * deleteCustomer/deleteService, callers do not need to handle a
 * "conflict" ApiError specially.
 */
export async function deleteDevice(id: string): Promise<void> {
  await apiFetch<void>(`/devices/${id}`, { method: 'DELETE' })
}
