import type { Device } from '@/types/device'
import type { DeviceModel } from '@/types/deviceModel'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'
import { listDeviceModels } from '@/services/deviceModels/deviceModelRepository'
import { listDeviceManufacturers } from '@/services/deviceManufacturers/deviceManufacturerRepository'
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
  device_model_id: string
  serial_number: string
  asset_tag: string
  status: Device['status']
  created_at: string
  updated_at: string
}

/**
 * Resolves a DeviceDto's device_model_id into the Device type's
 * read-only manufacturer/model display strings, by looking it up in
 * modelsById and, from there, its manufacturer in manufacturersById --
 * see types/device.ts's own doc comment for why Device carries both the
 * raw id and this resolved text. Falls back to "Unknown" rather than
 * throwing if a lookup misses: the two catalogs are fetched moments
 * earlier in the same request (see withDeviceCatalogs below), so a miss
 * here would mean a real data inconsistency, not a normal race --
 * surfacing a clearly-wrong label beats crashing the whole list over it.
 */
function fromDto(dto: DeviceDto, modelsById: Map<string, DeviceModel>, manufacturersById: Map<string, DeviceManufacturer>): Device {
  const model = modelsById.get(dto.device_model_id)
  const manufacturer = model ? manufacturersById.get(model.manufacturerId) : undefined

  return {
    id: dto.id,
    name: dto.name,
    description: dto.description,
    rackId: dto.rack_id,
    deviceModelId: dto.device_model_id,
    manufacturer: manufacturer?.name ?? 'Unknown',
    model: model?.name ?? 'Unknown',
    serialNumber: dto.serial_number,
    assetTag: dto.asset_tag,
    status: dto.status,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

/**
 * Fetches the Device Model and Device Manufacturer catalogs once and
 * returns lookup maps, for fromDto to join against -- both catalogs are
 * small (Administration-managed reference data, not per-device rows),
 * so fetching them fresh alongside every Device read is the same
 * "no server-side filtering, join client-side" tradeoff this codebase
 * makes throughout (see e.g. oltRepository.ts's
 * listOLTsByAccessNetworkId) rather than a real cost.
 */
async function fetchDeviceCatalogs(): Promise<{
  modelsById: Map<string, DeviceModel>
  manufacturersById: Map<string, DeviceManufacturer>
}> {
  const [models, manufacturers] = await Promise.all([listDeviceModels(), listDeviceManufacturers()])
  return {
    modelsById: new Map(models.map((m) => [m.id, m])),
    manufacturersById: new Map(manufacturers.map((m) => [m.id, m])),
  }
}

export interface DeviceListQuery {
  search?: string
  status?: Device['status'] | 'all'
  /**
   * When status is 'all' (no specific status picked), Retired devices
   * are excluded by default -- a deauthorized ONU
   * (internal/provisioning/kontron/service.DeauthorizationService marks
   * it Retired) should stop cluttering the default device list. Set
   * true to include them, or pick status: 'Retired' directly to see
   * only those.
   */
  includeRetired?: boolean
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
  const [{ devices }, { modelsById, manufacturersById }] = await Promise.all([
    apiFetch<{ devices: DeviceDto[] }>('/devices/'),
    fetchDeviceCatalogs(),
  ])
  return devices.map((dto) => fromDto(dto, modelsById, manufacturersById))
}

/** Returns every Device racked in the given Rack, for RackDetailView.vue's read-only Devices section. */
export async function listDevicesByRackId(rackId: string): Promise<Device[]> {
  const devices = await listAllDevices()
  return devices.filter((device) => device.rackId === rackId)
}

/** Fetches every Device and applies search/filter/sort/pagination client-side. */
export async function listDevices(query: DeviceListQuery = {}): Promise<DeviceListResult> {
  const {
    search = '',
    status = 'all',
    includeRetired = false,
    sortKey = 'name',
    sortDirection = 'asc',
    page = 1,
    pageSize = 15,
  } = query

  let results = (await listAllDevices()).filter((device) => matchesSearch(device, search))

  if (status !== 'all') {
    results = results.filter((device) => device.status === status)
  } else if (!includeRetired) {
    results = results.filter((device) => device.status !== 'Retired')
  }

  results = results.slice().sort(compareDevices(sortKey, sortDirection === 'desc' ? -1 : 1))

  const total = results.length
  const start = (page - 1) * pageSize
  return { items: results.slice(start, start + pageSize), total }
}

/** Fetches a single Device, returning null (not throwing) when it does not exist. */
export async function getDeviceById(id: string): Promise<Device | null> {
  try {
    const [dto, { modelsById, manufacturersById }] = await Promise.all([
      apiFetch<DeviceDto>(`/devices/${id}`),
      fetchDeviceCatalogs(),
    ])
    return fromDto(dto, modelsById, manufacturersById)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

/**
 * Fetches the Device with this exact serial number, returning null (not
 * throwing) when none exists -- the normal case for a newly-discovered
 * ONU, not an error. Used by DiscoverONUDialog.vue to avoid creating a
 * duplicate Device for a serial number already in inventory.
 */
export async function getDeviceBySerialNumber(serialNumber: string): Promise<Device | null> {
  try {
    const [dto, { modelsById, manufacturersById }] = await Promise.all([
      apiFetch<DeviceDto>(`/devices/by-serial-number/${encodeURIComponent(serialNumber)}`),
      fetchDeviceCatalogs(),
    ])
    return fromDto(dto, modelsById, manufacturersById)
  } catch (err) {
    if (err instanceof ApiError && err.kind === 'not_found') return null
    throw err
  }
}

export interface CreateDeviceInput {
  name: string
  deviceModelId: string
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
      device_model_id: input.deviceModelId,
      serial_number: input.serialNumber,
      asset_tag: input.assetTag,
      status: input.status,
      description: input.description,
      rack_id: input.rackId,
    },
  })
  const { modelsById, manufacturersById } = await fetchDeviceCatalogs()
  return fromDto(dto, modelsById, manufacturersById)
}

/**
 * Authorizes a physically-detected-but-unauthorized ONU on oltId/port
 * (see BlacklistedONU in types/onuDiagnostics.ts) and, only once that
 * succeeds, creates a real Device for it -- one action instead of the
 * two independently-skippable steps "Discover ONU" used to require
 * (authorize, then separately remember to fill out New Device). Lives
 * here rather than in provisioningRepository.ts because its result is
 * exactly a Device: same request shape as createDevice (see
 * CreateDeviceInput), same response DTO shape
 * (internal/provisioning/kontron/httpapi's authorizeAndCreateDeviceResponse
 * deliberately mirrors internal/inventory/httpapi's deviceResponse field
 * names), so it reuses fromDto/fetchDeviceCatalogs directly instead of
 * duplicating that join. DeviceFormDialog.vue calls this instead of
 * createDevice whenever its Serial Number field is a blacklist selection
 * rather than free text.
 */
export async function authorizeAndCreateDevice(oltId: string, port: string, input: CreateDeviceInput): Promise<Device> {
  const dto = await apiFetch<DeviceDto>(`/provisioning/olts/${oltId}/authorize-and-create-device`, {
    method: 'POST',
    body: {
      port,
      name: input.name,
      device_model_id: input.deviceModelId,
      serial_number: input.serialNumber,
      asset_tag: input.assetTag,
      status: input.status,
      description: input.description,
      rack_id: input.rackId,
    },
  })
  const { modelsById, manufacturersById } = await fetchDeviceCatalogs()
  return fromDto(dto, modelsById, manufacturersById)
}

export interface UpdateDeviceInput {
  name: string
  deviceModelId: string
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
      device_model_id: input.deviceModelId,
      serial_number: input.serialNumber,
      asset_tag: input.assetTag,
      status: input.status,
      description: input.description,
      rack_id: input.rackId,
    },
  })
  const { modelsById, manufacturersById } = await fetchDeviceCatalogs()
  return fromDto(dto, modelsById, manufacturersById)
}
