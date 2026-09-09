/**
 * The Device Manufacturer domain type, matching
 * internal/devicemanufacturer's real API shape exactly
 * (internal/devicemanufacturer/httpapi/dto.go's
 * deviceManufacturerResponse). An Administration-managed catalog entry
 * naming one manufacturer of Device hardware -- see
 * @/types/deviceModel's DeviceModel, which belongs to one of these.
 */
export interface DeviceManufacturer {
  id: string
  name: string
  description: string
  createdAt: string
  updatedAt: string
}
