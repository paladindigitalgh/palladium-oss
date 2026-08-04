import type { AssetStatus } from '@/types/customer'
import type { Device, DeviceStatus } from '@/types/device'
import { CUSTOMERS } from '@/services/customers/customerDataset'
import { createPrng, type Prng } from '@/lib/prng'

/**
 * The Device Collection Workspace's dataset: every managed device on the
 * live network, not an inventory of unassigned stock
 * (docs/09-WORKSPACE-SPECIFICATIONS.md section 10, "Device Workspace";
 * this milestone: "Do not treat Devices as Inventory"). Two sources, one
 * list:
 *
 * 1. Customer-premises devices -- read directly off
 *    services/customers/customerDataset.ts's existing Customer -> Service
 *    -> Asset tree (`customerDevices()` below). Nothing about a
 *    customer's identity is stored a second time: this file only *joins*
 *    the customer/service context onto each asset at generation time, the
 *    same way a real backend's device list endpoint would join across
 *    tables rather than duplicating columns.
 * 2. Network infrastructure devices (OLTs, aggregation switches) --
 *    these belong to a Site, not a Customer
 *    (docs/03-DOMAIN-MODEL.md section 8), so they are generated
 *    independently with no service/customer reference at all.
 *
 * Computed once at module load (own seeded PRNGs, independent of
 * customerDataset.ts's), exactly like CUSTOMERS -- stable across reloads,
 * never recomputed per query.
 */

const SERIAL_CHARS = '0123456789ABCDEF'

function serialFor(prng: Prng): string {
  let serial = ''
  for (let i = 0; i < 8; i += 1) {
    serial += SERIAL_CHARS[Math.floor(prng.random() * SERIAL_CHARS.length)]
  }
  return `SN-${serial}`
}

function mapAssetStatus(assetStatus: AssetStatus, prng: Prng): DeviceStatus {
  if (assetStatus === 'unknown') return 'provisioning'
  if (assetStatus === 'offline') return 'offline'
  return prng.chance(0.1) ? 'warning' : 'online'
}

/** Flattens every customer's service equipment into Device rows, joining in the owning customer/service context. */
function customerDevices(): Device[] {
  const prng = createPrng(24601)
  const devices: Device[] = []

  for (const customer of CUSTOMERS) {
    for (const service of customer.services) {
      for (const asset of service.equipment) {
        devices.push({
          id: asset.id,
          model: asset.model,
          serialNumber: asset.serialNumber,
          type: asset.role,
          technology: service.technology,
          status: mapAssetStatus(asset.status, prng),
          location: customer.city,
          serviceId: service.id,
          assignedCustomerId: customer.id,
          assignedCustomerName: customer.name,
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

/** Network infrastructure devices -- belong to a Site, never to a Customer. */
function infrastructureDevices(): Device[] {
  const prng = createPrng(560101)
  const devices: Device[] = []
  let oltIndex = 0
  let switchIndex = 0

  for (const site of SITES) {
    oltIndex += 1
    devices.push({
      id: `OLT-${1000 + oltIndex}`,
      model: prng.pick(OLT_MODELS),
      serialNumber: serialFor(prng),
      type: 'OLT',
      technology: prng.pickWeighted([['gpon', 3], ['xgs-pon', 2]] as const),
      status: prng.pickWeighted(INFRA_STATUS_WEIGHTS),
      location: site.name,
    })

    if (site.hasSwitch) {
      switchIndex += 1
      devices.push({
        id: `SW-${2000 + switchIndex}`,
        model: prng.pick(SWITCH_MODELS),
        serialNumber: serialFor(prng),
        type: 'Switch',
        status: prng.pickWeighted(INFRA_STATUS_WEIGHTS),
        location: site.name,
      })
    }
  }

  return devices
}

export const DEVICES: Device[] = [...customerDevices(), ...infrastructureDevices()]
