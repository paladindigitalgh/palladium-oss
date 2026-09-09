/**
 * The Device Model domain type, matching internal/devicemodel's real API
 * shape exactly (internal/devicemodel/httpapi/dto.go's
 * deviceModelResponse). An Administration-managed catalog entry naming
 * one hardware model made by the @/types/deviceManufacturer.DeviceManufacturer
 * identified by manufacturerId -- see @/types/device's Device, whose
 * deviceModelId references one of these in place of the free-text
 * Manufacturer/Model fields it used to carry directly.
 */
export interface DeviceModel {
  id: string
  manufacturerId: string
  name: string
  description: string
  createdAt: string
  updatedAt: string
}
