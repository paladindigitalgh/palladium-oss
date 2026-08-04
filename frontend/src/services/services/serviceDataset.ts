import type { Service, ServiceCategory } from '@/types/service'
import type { ServiceStatus } from '@/types/customer'
import type { Note } from '@/types/note'
import type { Device } from '@/types/device'
import type { HistoryEntry } from '@/lib/history'
import { CUSTOMERS } from '@/services/customers/customerDataset'
import { DEVICES } from '@/services/devices/deviceDataset'
import { createPrng } from '@/lib/prng'
import { NOW, addDays, daysBetween, formatAbsoluteDate, formatRelative, parseIsoDate } from '@/lib/dates'
import { finalizeHistory } from '@/lib/history'

/**
 * The Service Collection/Detail Workspace's dataset: every service owned
 * by a customer, flattened from the existing Customer -> Service tree
 * (services/customers/customerDataset.ts) exactly the way
 * services/devices/deviceDataset.ts flattens Customer -> Service ->
 * Asset into devices. `customerId`/`customerName`/`customerType` are
 * joined in, never a second stored copy of customer identity.
 *
 * Network fields (OLT, PON port, service VLAN) are read directly off the
 * service's own ONT device in DEVICES rather than generated
 * independently -- the same physical circuit should never disagree with
 * itself between the Service Detail and Device Detail Workspaces. This
 * is why this file imports deviceDataset.ts: customerDataset.ts (no
 * dependencies) -> deviceDataset.ts (depends on customers) ->
 * serviceDataset.ts (depends on both) is a one-way chain, never
 * circular.
 *
 * Computed once at module load, own seeded PRNG, exactly like CUSTOMERS
 * and DEVICES.
 */

function bandwidthProfileFor(tier: string): string {
  const match = tier.match(/^(\d+(?:\.\d+)?)\s*(Mbps|Gbps)/)
  if (!match) return 'Custom Bandwidth Profile'
  return `${match[1]} ${match[2]} / ${match[1]} ${match[2]} Symmetric`
}

const AUTH_PROFILES_STATIC = ['IPoE-Static-Standard']
const AUTH_PROFILES_DYNAMIC = ['IPoE-DHCP-Standard', 'PPPoE-RADIUS-Standard']

function isStaticCategory(category: ServiceCategory): boolean {
  return category === 'internet-static-ipv4' || category === 'business-internet'
}

function configVersionFor(prng: ReturnType<typeof createPrng>): string {
  return `v${prng.randomInt(1, 4)}.${prng.randomInt(0, 9)}.${prng.randomInt(0, 9)}`
}

function managementVlanFor(prng: ReturnType<typeof createPrng>): number {
  return prng.randomInt(10, 99)
}

function staticIpv4For(prng: ReturnType<typeof createPrng>): string {
  // RFC 5737 TEST-NET-3 -- reserved for documentation, never a real host.
  return `203.0.113.${prng.randomInt(10, 250)}`
}

function gatewayFor(ipv4: string): string {
  const parts = ipv4.split('.')
  return `${parts[0]}.${parts[1]}.${parts[2]}.1`
}

function ipv6For(prng: ReturnType<typeof createPrng>): string {
  // RFC 3849 documentation prefix (2001:db8::/32) -- never a real host.
  return `2001:db8:${prng.randomInt(1, 9999).toString(16)}::${prng.randomInt(1, 999).toString(16)}`
}

interface Addressing {
  ipv4Address?: string
  ipv6Address?: string
  gateway?: string
}

function networkAddressingFor(prng: ReturnType<typeof createPrng>, category: ServiceCategory): Addressing {
  const includesIpv6 = category === 'internet-ipv6' || (category === 'business-internet' && prng.chance(0.4))

  if (isStaticCategory(category)) {
    const ipv4Address = staticIpv4For(prng)
    return { ipv4Address, gateway: gatewayFor(ipv4Address), ipv6Address: includesIpv6 ? ipv6For(prng) : undefined }
  }

  return { ipv4Address: 'Dynamic (DHCP)', ipv6Address: includesIpv6 ? ipv6For(prng) : undefined }
}

function lastSyncFor(prng: ReturnType<typeof createPrng>, status: ServiceStatus): string {
  if (status === 'provisioning') return 'Not yet synchronized'
  if (status === 'active') return formatRelative(new Date(NOW.getTime() - prng.randomInt(1, 120) * 60000))
  return formatRelative(addDays(NOW, -prng.randomInt(1, 30)))
}

function activationDateFor(prng: ReturnType<typeof createPrng>, provisionedDate: Date, status: ServiceStatus): string | undefined {
  if (status === 'provisioning') return undefined
  return addDays(provisionedDate, prng.randomInt(0, 3)).toISOString().slice(0, 10)
}

const RESIDENTIAL_CATEGORY_WEIGHTS: [ServiceCategory, number][] = [
  ['internet', 70],
  ['internet-static-ipv4', 15],
  ['internet-ipv6', 15],
]

const BUSINESS_CATEGORY_WEIGHTS: [ServiceCategory, number][] = [
  ['business-internet', 65],
  ['internet-static-ipv4', 20],
  ['internet-ipv6', 15],
]

const SERVICE_EVENT_TEMPLATES = [
  { label: 'Configuration synchronized', description: 'Provisioning configuration synchronized with network state.' },
  { label: 'Bandwidth profile verified', description: 'Bandwidth profile confirmed against the provisioning system.' },
  { label: 'Authentication profile updated', description: 'Authentication profile refreshed during routine maintenance.' },
  { label: 'Diagnostics completed', description: 'Routine service diagnostic completed with no issues found.' },
  { label: 'IP address renewed', description: 'Dynamic IP address lease renewed.' },
  { label: 'VLAN reassigned', description: 'Service VLAN reassigned during network maintenance.' },
]

const SERVICE_ONBOARDING_TEMPLATES = [
  { label: 'Provisioning order received', description: 'Service order entered into the provisioning queue.' },
  { label: 'Profiles staged', description: 'Provisioning, bandwidth, and authentication profiles staged for activation.' },
]

function buildServiceHistory(
  prng: ReturnType<typeof createPrng>,
  serviceId: string,
  provisionedDate: Date,
  status: ServiceStatus,
  activationDate: string | undefined,
) {
  const entries: HistoryEntry[] = [
    { date: provisionedDate, template: { label: 'Service provisioned', description: 'Service record created and queued for activation.' } },
  ]

  if (status === 'provisioning') {
    const count = prng.randomInt(1, SERVICE_ONBOARDING_TEMPLATES.length)
    SERVICE_ONBOARDING_TEMPLATES.slice(0, count).forEach((template, index) => {
      entries.push({ date: addDays(provisionedDate, index + 1), template })
    })
  } else {
    if (activationDate) {
      entries.push({
        date: parseIsoDate(activationDate),
        template: { label: 'Service activated', description: 'Service activated and verified operational.' },
      })
    }

    const tenureDays = Math.max(1, daysBetween(provisionedDate, NOW))
    const eventCount = prng.randomInt(4, 10)
    for (let i = 0; i < eventCount; i += 1) {
      entries.push({ date: addDays(provisionedDate, prng.randomInt(1, tenureDays)), template: prng.pick(SERVICE_EVENT_TEMPLATES) })
    }

    if (status === 'suspended') {
      entries.push({
        date: addDays(NOW, -prng.randomInt(1, 30)),
        template: { label: 'Service suspended', description: 'Service suspended for non-payment.' },
      })
    }

    if (status === 'cancelled') {
      entries.push({
        date: addDays(NOW, -prng.randomInt(1, 60)),
        template: { label: 'Service cancelled', description: 'Service cancelled at customer request.' },
      })
    }
  }

  return finalizeHistory(serviceId, entries)
}

const NOTE_AUTHORS = ['Casey Whitmore', 'Jordan Reyes', 'Taylor Nguyen', 'Morgan Fields', 'Riley Osei', 'Alex Turner']

const SERVICE_NOTE_TEMPLATES = [
  "Confirmed bandwidth profile matches the customer's ordered tier.",
  'Customer requested a static IP; flagged for provisioning review.',
  'Verified authentication profile after a period of intermittent drops.',
  'Adjusted VLAN assignment during a network maintenance window.',
  'Customer confirmed service restored after scheduled maintenance.',
  'Flagged for profile review during next billing cycle.',
]

function buildServiceNotes(prng: ReturnType<typeof createPrng>, serviceId: string, provisionedDate: Date): Note[] {
  const count = prng.pickWeighted([[0, 5], [1, 3], [2, 2]] as const)
  const notes: Note[] = []
  for (let i = 0; i < count; i += 1) {
    const date = addDays(provisionedDate, prng.randomInt(0, Math.max(1, daysBetween(provisionedDate, NOW))))
    notes.push({
      id: `${serviceId}-NOTE-${i + 1}`,
      author: prng.pick(NOTE_AUTHORS),
      timestamp: formatAbsoluteDate(date),
      body: prng.pick(SERVICE_NOTE_TEMPLATES),
    })
  }
  return notes.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1))
}

function buildServices(): Service[] {
  const prng = createPrng(837211)

  const ontByServiceId = new Map<string, Device>()
  for (const device of DEVICES) {
    if (device.type === 'ONT' && device.serviceId) ontByServiceId.set(device.serviceId, device)
  }

  const services: Service[] = []

  for (const customer of CUSTOMERS) {
    for (const customerService of customer.services) {
      const ont = ontByServiceId.get(customerService.id)
      const provisionedDate = parseIsoDate(customerService.provisionedDate)
      const category = prng.pickWeighted(
        customer.type === 'residential' ? RESIDENTIAL_CATEGORY_WEIGHTS : BUSINESS_CATEGORY_WEIGHTS,
      )
      const activationDate = activationDateFor(prng, provisionedDate, customerService.status)
      const addressing = networkAddressingFor(prng, category)
      const { timeline, activity } = buildServiceHistory(
        prng,
        customerService.id,
        provisionedDate,
        customerService.status,
        activationDate,
      )

      services.push({
        id: customerService.id,
        tier: customerService.tier,
        technology: customerService.technology,
        status: customerService.status,
        category,
        serviceAddress: customerService.serviceAddress,
        provisionedDate: customerService.provisionedDate,
        activationDate,

        customerId: customer.id,
        customerName: customer.name,
        customerType: customer.type,

        provisioningProfile: ont?.configProfile ?? `${customerService.technology === 'gpon' ? 'GPON' : 'XGS'}-Standard`,
        bandwidthProfile: bandwidthProfileFor(customerService.tier),
        authenticationProfile: prng.pick(isStaticCategory(category) ? AUTH_PROFILES_STATIC : AUTH_PROFILES_DYNAMIC),
        configurationProfile: configVersionFor(prng),

        oltId: ont?.oltId,
        ponPort: ont?.ponPort,
        serviceVlan: ont?.serviceVlan,
        managementVlan: managementVlanFor(prng),
        ipv4Address: addressing.ipv4Address,
        ipv6Address: addressing.ipv6Address,
        gateway: addressing.gateway,

        lastSync: lastSyncFor(prng, customerService.status),

        activity,
        timeline,
        notes: buildServiceNotes(prng, customerService.id, provisionedDate),
      })
    }
  }

  return services
}

export const SERVICES: Service[] = buildServices()
