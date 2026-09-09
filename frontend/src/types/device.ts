/**
 * The Device domain type, matching internal/inventory's real API shape
 * exactly (internal/inventory/httpapi/dto.go's deviceResponse). Unlike
 * this file's previous mock version, nothing here is fabricated to fill
 * out a richer UI: the old telemetry-shaped fields (optical power,
 * temperature, uptime, management IP/VLANs, firmware version) had no
 * backend source and never will -- CLAUDE.md is explicit that Palladium
 * is NOT a monitoring platform ("Monitoring belongs in Zabbix or other
 * monitoring systems"). A Device here is a physical inventory record:
 * what it is, where it sits in the Rack hierarchy, and its lifecycle
 * status. Which Customer it is attached to, and which Service (if any)
 * it fulfills, are separate concerns -- see types/customerDevice.ts and
 * types/serviceEquipment.ts -- resolved on demand by the Device Detail
 * Workspace, never embedded here directly, but together they are
 * exactly what status tracks: this Device Collection is scoped to CPE
 * out in customer homes and businesses, not shelf/rack inventory, so
 * status only ever needs to say whether a Device is currently in use --
 * attached to a Customer directly, fulfilling a Service, or both (see
 * internal/inventory/device_status.go's own doc comment). Active/Unused
 * are set automatically, by internal/customerdevice/service.CustomerDeviceService
 * as a Device gains or loses an active CustomerDevice attachment and
 * internal/serviceequipment/service.ServiceEquipmentService as it gains
 * or loses an active ServiceEquipment record -- each checks the other's
 * before ever reverting to Unused, so losing one kind of attachment
 * while the other is still active correctly leaves it Active; a person
 * can still correct it by hand via Edit Device, but New Device no
 * longer offers Status as a choice at all, defaulting every new Device
 * to Unused (see DeviceFormDialog.vue).
 *
 * deviceModelId is the real, authoritative field (see
 * internal/inventory/model.go's Device doc comment): it references a
 * @/types/deviceModel.DeviceModel, itself belonging to a
 * @/types/deviceManufacturer.DeviceManufacturer, in place of what used
 * to be free-text Manufacturer/Model strings directly on Device.
 * manufacturer/model here are read-only display strings resolved by
 * joining those two catalogs -- see deviceRepository.ts's fromDto --
 * kept on this type so every existing display/search call site
 * (DeviceCollectionView.vue, DeviceDetailView.vue,
 * AssignServiceEquipmentDialog.vue, ServiceDetailView.vue) reads exactly
 * as it did before. Only DeviceFormDialog.vue writes deviceModelId
 * directly, via its Manufacturer/Model picker.
 */
export type DeviceStatus = 'Unused' | 'Active' | 'Retired'

export interface Device {
  id: string
  name: string
  description: string
  rackId: string | null
  deviceModelId: string
  manufacturer: string
  model: string
  serialNumber: string
  assetTag: string
  status: DeviceStatus
  createdAt: string
  updatedAt: string
}
