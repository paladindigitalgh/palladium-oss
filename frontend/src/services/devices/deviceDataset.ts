import type { AssetStatus } from '@/types/customer'
import type { ActivityEntry } from '@/types/activity'
import type { Note } from '@/types/note'
import type { Device, DeviceStatus } from '@/types/device'
import { CUSTOMERS } from '@/services/customers/customerDataset'
import { createPrng, type Prng } from '@/lib/prng'
import { NOW, addDays, daysBetween, formatIsoDate, formatAbsoluteDate, formatRelative, parseIsoDate } from '@/lib/dates'
import { finalizeHistory } from '@/lib/history'

/**
 * The Device Collection/Detail Workspace's dataset: every managed device
 * on the live network, not an inventory of unassigned stock
 * (docs/09-WORKSPACE-SPECIFICATIONS.md section 10, "Device Workspace";
 * "Do not treat Devices as Inventory"). Two sources, one list:
 *
 * 1. Customer-premises devices -- read directly off
 *    services/customers/customerDataset.ts's existing Customer -> Service
 *    -> Asset tree (`customerDevices()` below). Ownership (customer id
 *    and display name) is *joined in*, never a second stored copy;
 *    operational data (firmware, telemetry, history) is generated fresh
 *    for the device itself, since none of that lives on the equipment
 *    record and none of it is a real backend would keep as live
 *    operational state, not static equipment master data either.
 * 2. Network infrastructure devices (OLTs, aggregation switches) --
 *    belong to a Site, not a Customer (docs/03-DOMAIN-MODEL.md section
 *    8), generated independently with no service/customer reference.
 *
 * Infrastructure is generated first so customer-premises devices can
 * reference the OLT they actually home into (`oltBySite`) -- real
 * network topology, not an arbitrary per-device string.
 *
 * Computed once at module load (own seeded PRNGs, independent of
 * customerDataset.ts's), exactly like CUSTOMERS -- stable across reloads.
 */

const SERIAL_CHARS = '0123456789ABCDEF'

function serialFor(prng: Prng): string {
  let serial = ''
  for (let i = 0; i < 8; i += 1) {
    serial += SERIAL_CHARS[Math.floor(prng.random() * SERIAL_CHARS.length)]
  }
  return `SN-${serial}`
}

// The regional ISP standardizes on two fictional equipment brands
// (CLAUDE.md, "Plugin Philosophy": no real vendor names in core code) --
// one for optical gear, one for Ethernet gear, so a technician learns to
// associate a vendor with a device family rather than seeing one vendor
// per device.
const OPTICAL_VENDOR = 'Northpoint Networks'
const NETWORK_VENDOR = 'Cascade Networking'

function firmwareVersionFor(prng: Prng): string {
  return `${prng.randomInt(1, 6)}.${prng.randomInt(0, 20)}.${prng.randomInt(0, 9)}`
}

function configVersionFor(prng: Prng): string {
  return `v${prng.randomInt(1, 4)}.${prng.randomInt(0, 9)}.${prng.randomInt(0, 9)}`
}

function serviceVlanFor(prng: Prng): number {
  return prng.randomInt(100, 899)
}

function managementVlanFor(prng: Prng): number {
  return prng.randomInt(10, 99)
}

function managementIpFor(prng: Prng): string {
  return `10.${prng.randomInt(10, 99)}.${prng.randomInt(0, 255)}.${prng.randomInt(2, 254)}`
}

function ponPortFor(prng: Prng): string {
  return `${prng.randomInt(1, 4)}/${prng.randomInt(1, 8)}/${prng.randomInt(1, 64)}`
}

function temperatureFor(prng: Prng): number {
  return prng.randomInt(32, 58)
}

function ontOpticalPower(prng: Prng): number {
  return Math.round((-12 - prng.random() * 13) * 10) / 10
}

function oltOpticalPower(prng: Prng): number {
  return Math.round((1.5 + prng.random() * 3) * 10) / 10
}

function lastContactFor(prng: Prng, status: DeviceStatus): string {
  if (status === 'provisioning') return 'Not yet contacted'
  if (status === 'online') return formatRelative(new Date(NOW.getTime() - prng.randomInt(0, 14) * 60000))
  if (status === 'warning') return formatRelative(new Date(NOW.getTime() - prng.randomInt(1, 45) * 60000))
  return formatRelative(new Date(NOW.getTime() - prng.randomInt(2, 96) * 60 * 60000))
}

function uptimeSecondsFor(prng: Prng, status: DeviceStatus): number | undefined {
  if (status !== 'online' && status !== 'warning') return undefined
  return prng.randomInt(3600, 17280000)
}

function mapAssetStatus(assetStatus: AssetStatus, prng: Prng): DeviceStatus {
  if (assetStatus === 'unknown') return 'provisioning'
  if (assetStatus === 'offline') return 'offline'
  return prng.chance(0.1) ? 'warning' : 'online'
}

interface EventTemplate {
  kind: string
  label: string
  description: string
}

const DEVICE_EVENT_TEMPLATES: EventTemplate[] = [
  { kind: 'firmware', label: 'Firmware updated', description: 'Scheduled firmware update applied successfully.' },
  { kind: 'sync', label: 'Configuration synchronized', description: 'Provisioning configuration synchronized with network state.' },
  { kind: 'diagnostics', label: 'Diagnostics completed', description: 'Routine diagnostic check completed with no issues found.' },
  { kind: 'reboot', label: 'Device rebooted', description: 'Scheduled maintenance reboot completed.' },
  { kind: 'reregistered', label: 'Re-registered with network', description: 'Device re-established its session after a brief interruption.' },
  { kind: 'signal', label: 'Optical signal recalibrated', description: 'Optical transceiver levels recalibrated during routine maintenance.' },
  { kind: 'health', label: 'Health check completed', description: 'Automated health check completed successfully.' },
]

const DEVICE_ONBOARDING_TEMPLATES: EventTemplate[] = [
  { kind: 'staged', label: 'Configuration staged', description: 'Initial provisioning profile staged for activation.' },
  { kind: 'shipped', label: 'Device shipped to site', description: 'Equipment shipped for installation.' },
]

/** Builds a device's full operational history; Recent Activity is simply the newest slice of Timeline. */
function buildDeviceHistory(
  prng: Prng,
  deviceId: string,
  installedDate: Date,
  status: DeviceStatus,
): { timeline: ActivityEntry[]; activity: ActivityEntry[] } {
  const entries: { date: Date; template: EventTemplate }[] = [
    { date: installedDate, template: { kind: 'installed', label: 'Device installed', description: 'Device physically installed and connected to the network.' } },
  ]

  if (status === 'provisioning') {
    const count = prng.randomInt(1, DEVICE_ONBOARDING_TEMPLATES.length)
    DEVICE_ONBOARDING_TEMPLATES.slice(0, count).forEach((template, index) => {
      entries.push({ date: addDays(installedDate, index + 1), template })
    })
  } else {
    const tenureDays = Math.max(1, daysBetween(installedDate, NOW))
    const eventCount = prng.randomInt(5, 12)
    for (let i = 0; i < eventCount; i += 1) {
      entries.push({ date: addDays(installedDate, prng.randomInt(1, tenureDays)), template: prng.pick(DEVICE_EVENT_TEMPLATES) })
    }
    if (status === 'offline') {
      entries.push({
        date: addDays(NOW, -prng.randomInt(0, 3)),
        template: { kind: 'lost-contact', label: 'Lost contact', description: 'Device stopped responding to polling.' },
      })
    }
  }

  return finalizeHistory(deviceId, entries)
}

const NOTE_AUTHORS = ['Casey Whitmore', 'Jordan Reyes', 'Taylor Nguyen', 'Morgan Fields', 'Riley Osei', 'Alex Turner']

const DEVICE_NOTE_TEMPLATES = [
  'Confirmed optical levels within spec during last truck roll.',
  'Customer reported intermittent drops; recommend checking the fiber connector for contamination.',
  'Relocated to a ventilated shelf during last site visit to reduce operating temperature.',
  'Firmware update deferred at customer request until after business hours.',
  'Spare unit staged at the warehouse in case of failure.',
  'Verified serial number against the work order during installation.',
  'A reset resolved a temporary loss of sync; no further action needed.',
  'Flagged for firmware upgrade during the next maintenance window.',
]

function buildDeviceNotes(prng: Prng, deviceId: string, installedDate: Date): Note[] {
  const count = prng.pickWeighted([[0, 5], [1, 3], [2, 2]] as const)
  const notes: Note[] = []
  for (let i = 0; i < count; i += 1) {
    const date = addDays(installedDate, prng.randomInt(0, Math.max(1, daysBetween(installedDate, NOW))))
    notes.push({
      id: `${deviceId}-NOTE-${i + 1}`,
      author: prng.pick(NOTE_AUTHORS),
      timestamp: formatAbsoluteDate(date),
      body: prng.pick(DEVICE_NOTE_TEMPLATES),
    })
  }
  return notes.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1))
}

const ONT_PROFILES_GPON = ['GPON-Residential-Standard', 'GPON-Business-Standard']
const ONT_PROFILES_XGS = ['XGS-Residential-Premium', 'XGS-Business-Premium']
const ROUTER_PROFILES = ['CPE-Router-Standard', 'CPE-Router-Business']
const OLT_PROFILES = ['Core-OLT-Default']
const SWITCH_PROFILES = ['Core-Switch-Aggregation']

// Maps every customer town onto the Site that actually serves it, so a
// customer's ONT can reference a real, consistent OLT rather than a
// disconnected random string.
const TOWN_TO_SITE: Record<string, string> = {
  'Idaho Falls': 'Idaho Falls Central Office',
  Ammon: 'Idaho Falls Central Office',
  Iona: 'Idaho Falls Central Office',
  Rigby: 'Rigby Remote Cabinet',
  Ririe: 'Rigby Remote Cabinet',
  Rexburg: 'Rexburg Central Office',
  Menan: 'Rexburg Central Office',
  Blackfoot: 'Blackfoot POP',
  Shelley: 'Shelley Remote Cabinet',
  Ucon: 'Ucon Remote Cabinet',
}

/** Flattens every customer's service equipment into Device rows, joining in the owning customer/service context. */
function customerDevices(oltBySite: Map<string, string>): Device[] {
  const prng = createPrng(24601)
  const devices: Device[] = []

  for (const customer of CUSTOMERS) {
    const siteName = TOWN_TO_SITE[customer.city] ?? SITES[0].name
    const oltId = oltBySite.get(siteName)

    for (const service of customer.services) {
      const installedDate = parseIsoDate(service.provisionedDate)

      for (const asset of service.equipment) {
        const status = mapAssetStatus(asset.status, prng)
        const isOnt = asset.role === 'ONT'
        const { timeline, activity } = buildDeviceHistory(prng, asset.id, installedDate, status)

        devices.push({
          id: asset.id,
          model: asset.model,
          serialNumber: asset.serialNumber,
          type: asset.role,
          technology: service.technology,
          status,
          location: customer.city,
          serviceId: service.id,
          assignedCustomerId: customer.id,
          assignedCustomerName: customer.name,

          vendor: isOnt ? OPTICAL_VENDOR : NETWORK_VENDOR,
          firmwareVersion: firmwareVersionFor(prng),
          installedDate: service.provisionedDate,

          siteName,
          oltId: isOnt ? oltId : undefined,
          ponPort: isOnt ? ponPortFor(prng) : undefined,
          managementIp: isOnt ? undefined : managementIpFor(prng),

          lastContact: lastContactFor(prng, status),
          uptimeSeconds: uptimeSecondsFor(prng, status),
          opticalPowerDbm: isOnt && status !== 'provisioning' ? ontOpticalPower(prng) : undefined,
          temperatureC: temperatureFor(prng),

          configProfile: isOnt
            ? prng.pick(service.technology === 'gpon' ? ONT_PROFILES_GPON : ONT_PROFILES_XGS)
            : prng.pick(ROUTER_PROFILES),
          serviceVlan: serviceVlanFor(prng),
          managementVlan: isOnt ? undefined : managementVlanFor(prng),
          configVersion: configVersionFor(prng),

          activity,
          timeline,
          notes: buildDeviceNotes(prng, asset.id, installedDate),
        })
      }
    }
  }

  return devices
}

interface Site {
  name: string
  hasSwitch: boolean
}

const SITES: Site[] = [
  { name: 'Idaho Falls Central Office', hasSwitch: true },
  { name: 'Rexburg Central Office', hasSwitch: true },
  { name: 'Ammon POP', hasSwitch: false },
  { name: 'Rigby Remote Cabinet', hasSwitch: false },
  { name: 'Blackfoot POP', hasSwitch: false },
  { name: 'Shelley Remote Cabinet', hasSwitch: false },
  { name: 'Ucon Remote Cabinet', hasSwitch: false },
]

const OLT_MODELS = ['OLT Chassis Model OC-800', 'OLT Chassis Model OC-1200']
const SWITCH_MODELS = ['Aggregation Switch Model AS-24', 'Aggregation Switch Model AS-48']

const INFRA_STATUS_WEIGHTS: [DeviceStatus, number][] = [
  ['online', 85],
  ['warning', 8],
  ['offline', 4],
  ['provisioning', 3],
]

/** Network infrastructure devices -- belong to a Site, never to a Customer. Generated first so customerDevices() can reference a real OLT per site. */
function infrastructureDevices(): { devices: Device[]; oltBySite: Map<string, string> } {
  const prng = createPrng(560101)
  const devices: Device[] = []
  const oltBySite = new Map<string, string>()
  let oltIndex = 0
  let switchIndex = 0

  for (const site of SITES) {
    oltIndex += 1
    const oltId = `OLT-${1000 + oltIndex}`
    oltBySite.set(site.name, oltId)

    const oltInstalled = addDays(NOW, -prng.randomInt(200, 3000))
    const oltStatus = prng.pickWeighted(INFRA_STATUS_WEIGHTS)
    const oltHistory = buildDeviceHistory(prng, oltId, oltInstalled, oltStatus)

    devices.push({
      id: oltId,
      model: prng.pick(OLT_MODELS),
      serialNumber: serialFor(prng),
      type: 'OLT',
      technology: prng.pickWeighted([['gpon', 3], ['xgs-pon', 2]] as const),
      status: oltStatus,
      location: site.name,

      vendor: OPTICAL_VENDOR,
      firmwareVersion: firmwareVersionFor(prng),
      installedDate: formatIsoDate(oltInstalled),

      siteName: site.name,
      managementIp: managementIpFor(prng),
      uplinkPort: `Te1/1/${prng.randomInt(1, 4)}`,

      lastContact: lastContactFor(prng, oltStatus),
      uptimeSeconds: uptimeSecondsFor(prng, oltStatus),
      opticalPowerDbm: oltStatus === 'provisioning' ? undefined : oltOpticalPower(prng),
      temperatureC: temperatureFor(prng),

      configProfile: prng.pick(OLT_PROFILES),
      managementVlan: managementVlanFor(prng),
      configVersion: configVersionFor(prng),

      activity: oltHistory.activity,
      timeline: oltHistory.timeline,
      notes: buildDeviceNotes(prng, oltId, oltInstalled),
    })

    if (site.hasSwitch) {
      switchIndex += 1
      const switchId = `SW-${2000 + switchIndex}`
      const switchInstalled = addDays(NOW, -prng.randomInt(200, 3000))
      const switchStatus = prng.pickWeighted(INFRA_STATUS_WEIGHTS)
      const switchHistory = buildDeviceHistory(prng, switchId, switchInstalled, switchStatus)

      devices.push({
        id: switchId,
        model: prng.pick(SWITCH_MODELS),
        serialNumber: serialFor(prng),
        type: 'Switch',
        status: switchStatus,
        location: site.name,

        vendor: NETWORK_VENDOR,
        firmwareVersion: firmwareVersionFor(prng),
        installedDate: formatIsoDate(switchInstalled),

        siteName: site.name,
        managementIp: managementIpFor(prng),
        uplinkPort: `Te1/0/${prng.randomInt(1, 2)}`,

        lastContact: lastContactFor(prng, switchStatus),
        uptimeSeconds: uptimeSecondsFor(prng, switchStatus),
        temperatureC: temperatureFor(prng),

        configProfile: prng.pick(SWITCH_PROFILES),
        managementVlan: managementVlanFor(prng),
        configVersion: configVersionFor(prng),

        activity: switchHistory.activity,
        timeline: switchHistory.timeline,
        notes: buildDeviceNotes(prng, switchId, switchInstalled),
      })
    }
  }

  return { devices, oltBySite }
}

const { devices: INFRASTRUCTURE_DEVICES, oltBySite: OLT_BY_SITE } = infrastructureDevices()

export const DEVICES: Device[] = [...customerDevices(OLT_BY_SITE), ...INFRASTRUCTURE_DEVICES]
