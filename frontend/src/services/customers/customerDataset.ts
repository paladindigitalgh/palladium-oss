import type {
  Customer,
  CustomerAlert,
  CustomerActivityEntry,
  CustomerAsset,
  CustomerContact,
  CustomerNote,
  CustomerService,
  CustomerStatus,
  CustomerType,
  ServiceStatus,
  ServiceTechnology,
} from '@/types/customer'

/**
 * A development dataset simulating a believable regional fiber ISP
 * serving eastern Idaho, generated once from a seeded PRNG so the
 * dataset is stable across dev-server reloads instead of reshuffling on
 * every hot reload. Private to the service layer -- components never
 * import this file directly, only customerRepository.ts (see that
 * file's own doc comment).
 */

// mulberry32: small, dependency-free, deterministic PRNG. Only used to
// keep this fixture dataset stable; never used anywhere security-sensitive.
function mulberry32(seed: number) {
  let state = seed
  return function random() {
    state |= 0
    state = (state + 0x6d2b79f5) | 0
    let t = Math.imul(state ^ (state >>> 15), 1 | state)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

const random = mulberry32(133742)

function pick<T>(items: readonly T[]): T {
  return items[Math.floor(random() * items.length)]
}

function pickWeighted<T>(items: readonly [T, number][]): T {
  const total = items.reduce((sum, [, weight]) => sum + weight, 0)
  let roll = random() * total
  for (const [item, weight] of items) {
    roll -= weight
    if (roll <= 0) return item
  }
  return items[items.length - 1][0]
}

function randomInt(min: number, max: number): number {
  return Math.floor(random() * (max - min + 1)) + min
}

function chance(probability: number): boolean {
  return random() < probability
}

// Fixed rather than Date.now(): keeps generated dates (and their
// relative-time labels) deterministic across sessions instead of
// silently drifting each day the dev server happens to run.
const NOW = new Date(2026, 7, 4)

const MALE_FIRST_NAMES = [
  'James', 'Michael', 'Robert', 'John', 'David', 'William', 'Richard', 'Joseph', 'Thomas',
  'Christopher', 'Daniel', 'Matthew', 'Anthony', 'Mark', 'Steven', 'Paul', 'Andrew', 'Joshua',
  'Kenneth', 'Kevin', 'Brian', 'Jason', 'Timothy', 'Ryan', 'Jacob', 'Gary', 'Nicholas', 'Eric',
  'Jonathan', 'Justin', 'Scott', 'Brandon', 'Benjamin', 'Samuel', 'Gregory', 'Alexander',
  'Patrick', 'Jack', 'Tyler', 'Aaron', 'Adam', 'Nathan', 'Henry', 'Douglas', 'Zachary', 'Peter',
  'Spencer', 'Wyatt', 'Colton', 'Bridger',
]

const FEMALE_FIRST_NAMES = [
  'Mary', 'Patricia', 'Jennifer', 'Linda', 'Elizabeth', 'Barbara', 'Susan', 'Jessica', 'Sarah',
  'Karen', 'Lisa', 'Nancy', 'Margaret', 'Sandra', 'Ashley', 'Kimberly', 'Emily', 'Donna',
  'Michelle', 'Carol', 'Amanda', 'Melissa', 'Deborah', 'Stephanie', 'Rebecca', 'Laura', 'Sharon',
  'Cynthia', 'Kathleen', 'Amy', 'Angela', 'Helen', 'Anna', 'Brenda', 'Pamela', 'Nicole', 'Emma',
  'Samantha', 'Katherine', 'Christine', 'Rachel', 'Catherine', 'Carolyn', 'Janet', 'Maria',
  'Heather', 'Diane', 'Madison', 'Brooklyn', 'Kinsley',
]

const LAST_NAMES = [
  'Smith', 'Johnson', 'Williams', 'Brown', 'Jones', 'Garcia', 'Miller', 'Davis', 'Rodriguez',
  'Martinez', 'Hernandez', 'Lopez', 'Wilson', 'Anderson', 'Thomas', 'Taylor', 'Moore', 'Jackson',
  'Martin', 'Lee', 'Thompson', 'White', 'Harris', 'Clark', 'Lewis', 'Robinson', 'Walker', 'Young',
  'Allen', 'King', 'Wright', 'Scott', 'Torres', 'Hill', 'Green', 'Adams', 'Nelson', 'Baker',
  'Hall', 'Campbell', 'Mitchell', 'Carter', 'Roberts', 'Christensen', 'Larsen', 'Hansen', 'Olsen',
  'Jensen', 'Andersen', 'Nielsen', 'Petersen', 'Rasmussen', 'Sorensen', 'Whitmore', 'Hobbs',
  'Call', 'Pratt', 'Cazier', 'Erickson', 'Bagley', 'Fillmore', 'Hurd', 'Openshaw',
]

const TOWNS: { city: string; postalCodes: string[] }[] = [
  { city: 'Idaho Falls', postalCodes: ['83401', '83402', '83404', '83406'] },
  { city: 'Ammon', postalCodes: ['83406'] },
  { city: 'Rigby', postalCodes: ['83442'] },
  { city: 'Shelley', postalCodes: ['83274'] },
  { city: 'Blackfoot', postalCodes: ['83221'] },
  { city: 'Ucon', postalCodes: ['83454'] },
  { city: 'Iona', postalCodes: ['83427'] },
  { city: 'Rexburg', postalCodes: ['83440'] },
  { city: 'Menan', postalCodes: ['83434'] },
  { city: 'Ririe', postalCodes: ['83443'] },
]

const STREET_NAMES = [
  'Elm', 'Maple', 'Sunset', 'River', 'Ridge', 'Canyon', 'Meadow', 'Aspen', 'Birch', 'Cedar',
  'Pioneer', 'Teton', 'Snake River', 'Sagebrush', 'Cottonwood', 'Willow', 'Foothill', 'Grove',
  'Sunrise', 'Bonneville', 'Sawtooth', 'Northgate', 'Harvest', 'Butte',
]

const STREET_SUFFIXES = ['St', 'Ave', 'Dr', 'Ln', 'Way', 'Ct', 'Rd', 'Blvd', 'Cir']

const BUSINESS_PREFIXES = [
  'Snake River', 'Teton Ridge', 'Canyon Rim', 'Cottonwood', 'Willow Creek', 'Northgate',
  'Sawtooth', 'Blue Sky', 'Harvest Moon', 'Summit', 'Bonneville', 'Falls Valley', 'Highline',
  'Riverbend', 'Sagebrush', 'Foothill', 'Pioneer', 'Sunrise', 'Butte County', 'Meadowlark',
]

const BUSINESS_SUFFIXES = [
  'Dental', 'Family Dentistry', 'Hardware & Supply', 'Veterinary Clinic', 'Grain Cooperative',
  'Auto Body', 'Feed & Seed', 'Excavation', 'Contractors', 'Insurance Group', 'Legal Group',
  'Coffee Roasters', 'Trucking', 'Storage', 'Bakery', 'Auto Repair', 'Property Management',
  'Orthodontics', 'Realty', 'Landscaping', 'Electric', 'Plumbing', 'Diner', 'Physical Therapy',
  'Veterinary Hospital', 'Body Shop', 'Machine Shop',
]

const BUSINESS_ROLES = ['Owner', 'General Manager', 'Office Manager', 'Operations Manager']
const SECONDARY_RESIDENTIAL_ROLES = ['Household Member', 'Spouse']
const SECONDARY_BUSINESS_ROLES = ['Assistant Manager', 'IT Contact', 'Accounts Payable']

const RESIDENTIAL_SERVICES: [{ tier: string; technology: ServiceTechnology }, number][] = [
  [{ tier: '100 Mbps Residential Fiber', technology: 'gpon' }, 5],
  [{ tier: '250 Mbps Residential Fiber', technology: 'gpon' }, 6],
  [{ tier: '500 Mbps Residential Fiber', technology: 'gpon' }, 4],
  [{ tier: '1 Gbps Residential Fiber', technology: 'gpon' }, 3],
  [{ tier: '2 Gbps Residential Fiber', technology: 'xgs-pon' }, 2],
  [{ tier: '5 Gbps Residential Fiber', technology: 'xgs-pon' }, 1],
]

const BUSINESS_SERVICES: [{ tier: string; technology: ServiceTechnology }, number][] = [
  [{ tier: '50 Mbps Business Fiber', technology: 'gpon' }, 3],
  [{ tier: '250 Mbps Business Fiber', technology: 'gpon' }, 4],
  [{ tier: '1 Gbps Business Fiber', technology: 'xgs-pon' }, 4],
  [{ tier: '5 Gbps Business Fiber', technology: 'xgs-pon' }, 2],
  [{ tier: '10 Gbps Business Fiber', technology: 'xgs-pon' }, 1],
]

const SECONDARY_BUSINESS_SERVICES: [{ tier: string; technology: ServiceTechnology }, number][] = [
  [{ tier: 'Static IP Block (5)', technology: 'gpon' }, 2],
  [{ tier: 'Backup Business Fiber - 50 Mbps', technology: 'gpon' }, 2],
  [{ tier: 'Point-to-Point Transport Circuit', technology: 'xgs-pon' }, 1],
]

const GPON_MODELS = ['ONT Model GN-100', 'ONT Model GN-110', 'ONT Model GN-120']
const XGS_MODELS = ['ONT Model XG-400', 'ONT Model XG-410', 'ONT Model XG-420']
const ROUTER_MODELS = ['Router Model RT-210', 'Router Model RT-220', 'Router Model RT-310']

const STATUS_WEIGHTS: [CustomerStatus, number][] = [
  ['active', 68],
  ['pending', 12],
  ['suspended', 11],
  ['cancelled', 9],
]

const CUSTOMER_TYPE_WEIGHTS: [CustomerType, number][] = [
  ['residential', 68],
  ['business', 32],
]

function daysBetween(a: Date, b: Date): number {
  return Math.round((b.getTime() - a.getTime()) / (1000 * 60 * 60 * 24))
}

function addDays(date: Date, days: number): Date {
  return new Date(date.getTime() + days * 24 * 60 * 60 * 1000)
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

function formatAbsoluteDate(date: Date): string {
  return `${MONTHS[date.getMonth()]} ${date.getDate()}, ${date.getFullYear()}`
}

function formatIsoDate(date: Date): string {
  return date.toISOString().slice(0, 10)
}

function formatRelative(date: Date): string {
  const minutes = Math.floor((NOW.getTime() - date.getTime()) / 60000)
  if (minutes < 1) return 'Just now'
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'} ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  const days = Math.floor(hours / 24)
  if (days < 2) return 'Yesterday'
  if (days < 30) return `${days} days ago`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} month${months === 1 ? '' : 's'} ago`
  return formatAbsoluteDate(date)
}

function randomInstallDate(): Date {
  // Between 2016-01-01 and 2026-06-01, before this milestone's "today".
  const start = new Date(2016, 0, 1).getTime()
  const end = new Date(2026, 5, 1).getTime()
  return new Date(start + random() * (end - start))
}

/** A pending customer is still onboarding -- its install date should read as recent, not years old. */
function recentInstallDate(): Date {
  return addDays(NOW, -randomInt(1, 45))
}

function residentialName(): string {
  const first = chance(0.5) ? pick(MALE_FIRST_NAMES) : pick(FEMALE_FIRST_NAMES)
  return `${first} ${pick(LAST_NAMES)}`
}

function businessName(): string {
  return `${pick(BUSINESS_PREFIXES)} ${pick(BUSINESS_SUFFIXES)}`
}

function streetAddress(): string {
  return `${randomInt(100, 9899)} ${pick(STREET_NAMES)} ${pick(STREET_SUFFIXES)}`
}

function slugify(text: string): string {
  return text.toLowerCase().replace(/[^a-z0-9]+/g, '')
}

function emailFor(name: string, domain: string): string {
  const parts = name.toLowerCase().split(' ')
  const first = parts[0]
  const last = parts[parts.length - 1]
  return `${first}.${last}@${domain}`
}

function randomPhone(): string {
  return `(208) ${randomInt(200, 999)}-${String(randomInt(0, 9999)).padStart(4, '0')}`
}

const SERIAL_CHARS = '0123456789ABCDEF'

function randomSerial(): string {
  let serial = ''
  for (let i = 0; i < 8; i += 1) {
    serial += SERIAL_CHARS[Math.floor(random() * SERIAL_CHARS.length)]
  }
  return `SN-${serial}`
}

function modelFor(role: 'ONU' | 'Router', technology: ServiceTechnology): string {
  if (role === 'Router') return pick(ROUTER_MODELS)
  return pick(technology === 'gpon' ? GPON_MODELS : XGS_MODELS)
}

function serviceStatusFor(customerStatus: CustomerStatus): ServiceStatus {
  switch (customerStatus) {
    case 'active':
      return 'active'
    case 'suspended':
      return 'suspended'
    case 'pending':
      return 'pending'
    case 'cancelled':
      return 'decommissioned'
  }
}

function assetStatusFor(serviceStatus: ServiceStatus): CustomerAsset['status'] {
  if (serviceStatus === 'pending') return 'unknown'
  if (serviceStatus === 'active') return chance(0.92) ? 'online' : 'offline'
  return 'offline'
}

function buildAsset(idPrefix: string, index: number, role: 'ONU' | 'Router', technology: ServiceTechnology, serviceStatus: ServiceStatus): CustomerAsset {
  return {
    id: `${idPrefix}-AST-${index}`,
    role,
    model: modelFor(role, technology),
    serialNumber: randomSerial(),
    status: assetStatusFor(serviceStatus),
  }
}

function buildService(
  customerId: string,
  index: number,
  type: CustomerType,
  status: ServiceStatus,
  provisionedDate: Date,
  serviceAddress: string,
  pool: readonly [{ tier: string; technology: ServiceTechnology }, number][],
): CustomerService {
  const picked = pickWeighted(pool)
  const idPrefix = `${customerId}-SVC-${index}`
  const equipment = [buildAsset(idPrefix, 1, 'ONU', picked.technology, status)]
  if (type === 'business' && chance(0.35)) {
    equipment.push(buildAsset(idPrefix, 2, 'Router', picked.technology, status))
  }

  return {
    id: idPrefix,
    tier: picked.tier,
    technology: picked.technology,
    status,
    provisionedDate: formatIsoDate(provisionedDate),
    serviceAddress,
    equipment,
  }
}

function buildContacts(type: CustomerType, customerName: string, emailDomain: string): Customer['contacts'] {
  const primary: CustomerContact =
    type === 'residential'
      ? { name: customerName, phone: randomPhone(), email: emailFor(customerName, emailDomain) }
      : {
          name: residentialName(),
          role: pick(BUSINESS_ROLES),
          phone: randomPhone(),
          email: emailFor(residentialName(), emailDomain),
        }

  if (!chance(0.3)) {
    return { primary }
  }

  const secondaryName = residentialName()
  const secondary: CustomerContact = {
    name: secondaryName,
    role: type === 'residential' ? pick(SECONDARY_RESIDENTIAL_ROLES) : pick(SECONDARY_BUSINESS_ROLES),
    phone: randomPhone(),
    email: emailFor(secondaryName, emailDomain),
  }

  return { primary, secondary }
}

const ALERT_TEMPLATES: Record<'critical' | 'warning' | 'info', { title: string; description: string }[]> = {
  critical: [
    { title: 'ONU Offline', description: 'This optical network unit has not reported in over 24 hours.' },
    { title: 'Service Outage', description: 'No traffic has been observed on this service since the last polling cycle.' },
  ],
  warning: [
    { title: 'Degraded Optical Signal', description: 'Received optical power is below the recommended threshold.' },
    { title: 'High Latency Detected', description: 'Recent diagnostics show latency above the normal operating range.' },
    { title: 'Repeated Reconnects', description: 'This ONU has re-registered with the OLT multiple times in the last 24 hours.' },
  ],
  info: [
    { title: 'Firmware Update Available', description: 'A newer firmware version is available for this ONU model.' },
    { title: 'Installation Pending', description: 'Awaiting technician dispatch to complete installation.' },
    { title: 'Non-Payment Hold', description: 'Service suspended pending payment resolution.' },
  ],
}

function buildAlerts(customerId: string, status: CustomerStatus): CustomerAlert[] {
  if (status === 'cancelled') return []

  let chanceOfAlert = 0.1
  if (status === 'suspended') chanceOfAlert = 0.7
  if (status === 'pending') chanceOfAlert = 0.55

  if (!chance(chanceOfAlert)) return []

  const severity =
    status === 'suspended' || status === 'pending' ? 'info' : pickWeighted([['warning', 3], ['critical', 1], ['info', 2]] as const)
  const template = pick(ALERT_TEMPLATES[severity])
  const alerts: CustomerAlert[] = [
    {
      id: `${customerId}-ALRT-1`,
      severity,
      title: template.title,
      description: template.description,
      timestamp: formatRelative(addDays(NOW, -randomInt(0, 7))),
    },
  ]

  if (chance(0.15)) {
    const secondSeverity = pickWeighted([['warning', 3], ['critical', 1], ['info', 2]] as const)
    const secondTemplate = pick(ALERT_TEMPLATES[secondSeverity])
    alerts.push({
      id: `${customerId}-ALRT-2`,
      severity: secondSeverity,
      title: secondTemplate.title,
      description: secondTemplate.description,
      timestamp: formatRelative(addDays(NOW, -randomInt(0, 14))),
    })
  }

  return alerts
}

interface EventTemplate {
  kind: string
  label: string
  description: string
}

function operationalEventTemplates(primaryTier: string): EventTemplate[] {
  return [
    { kind: 'installed', label: 'Service installed', description: 'Technician completed on-site installation.' },
    { kind: 'activated', label: 'Service activated', description: 'Service verified operational after installation.' },
    { kind: 'firmware', label: 'ONU firmware updated', description: 'Scheduled firmware update applied successfully.' },
    { kind: 'sync', label: 'Configuration synchronized', description: 'Provisioning configuration synchronized with network state.' },
    { kind: 'diagnostics', label: 'Diagnostics completed', description: 'Routine diagnostic check completed with no issues found.' },
    { kind: 'speed-check', label: 'Speed test completed', description: 'Speed test confirmed provisioned throughput.' },
    { kind: 'onu-replaced', label: 'ONU replaced', description: 'Optical network unit replaced due to hardware fault.' },
    { kind: 'support-note', label: 'Support note logged', description: 'Operator logged a note following a customer contact.' },
    {
      kind: 'upgraded',
      label: `Service upgraded to ${primaryTier}`,
      description: 'Speed tier upgraded at customer request.',
    },
  ]
}

const ONBOARDING_TEMPLATES: EventTemplate[] = [
  { kind: 'order', label: 'Service order received', description: 'New service order entered into the provisioning queue.' },
  { kind: 'survey', label: 'Site survey scheduled', description: 'Site survey scheduled to confirm serviceability.' },
  { kind: 'install-scheduled', label: 'Installation scheduled', description: 'Technician dispatch scheduled for installation.' },
]

/**
 * Builds a customer's full operational history and returns both the
 * long-form Timeline and the Recent Activity slice at the front of it --
 * Recent Activity is deliberately not an independent list, it is simply
 * the newest few Timeline entries, matching how an activity feed relates
 * to a full audit log in a real operational system.
 */
function buildOperationalHistory(
  customerId: string,
  status: CustomerStatus,
  installDate: Date,
  primaryTier: string,
): { timeline: CustomerActivityEntry[]; activity: CustomerActivityEntry[] } {
  const provisioned: { date: Date; template: EventTemplate } = {
    date: installDate,
    template: { kind: 'provisioned', label: 'Customer provisioned', description: 'Account created and service scheduled for installation.' },
  }

  const entries: { date: Date; template: EventTemplate }[] = [provisioned]

  if (status === 'pending') {
    // A pending customer has onboarding history only -- nothing to
    // service yet.
    const onboardingCount = randomInt(1, ONBOARDING_TEMPLATES.length)
    const shuffled = ONBOARDING_TEMPLATES.slice(0, onboardingCount)
    shuffled.forEach((template, index) => {
      entries.push({ date: addDays(installDate, index + 1), template })
    })
  } else {
    const tenureDays = Math.max(1, daysBetween(installDate, NOW))
    const eventCount = randomInt(6, 13)
    const templates = operationalEventTemplates(primaryTier)
    for (let i = 0; i < eventCount; i += 1) {
      const offset = randomInt(1, tenureDays)
      entries.push({ date: addDays(installDate, offset), template: pick(templates) })
    }

    if (status === 'suspended') {
      entries.push({
        date: addDays(NOW, -randomInt(1, 30)),
        template: { kind: 'suspended', label: 'Service suspended', description: 'Service suspended for non-payment.' },
      })
    }

    if (status === 'cancelled') {
      entries.push({
        date: addDays(NOW, -randomInt(1, 60)),
        template: { kind: 'cancelled', label: 'Service cancelled', description: 'Account closed at customer request.' },
      })
    }
  }

  entries.sort((a, b) => b.date.getTime() - a.date.getTime())

  const timeline: CustomerActivityEntry[] = entries.map((entry, index) => ({
    id: `${customerId}-EVT-${index + 1}`,
    label: entry.template.label,
    timestamp: formatAbsoluteDate(entry.date),
    description: entry.template.description,
  }))

  const activity: CustomerActivityEntry[] = entries.slice(0, Math.min(8, entries.length)).map((entry, index) => ({
    id: `${customerId}-ACT-${index + 1}`,
    label: entry.template.label,
    timestamp: formatRelative(entry.date),
    description: entry.template.description,
  }))

  return { timeline, activity }
}

const NOTE_TEMPLATES = [
  'Customer requested email notifications for outages. Added to notification list.',
  'Confirmed a static IP is not required at this time.',
  'Customer reported occasional buffering during peak hours; recommended relocating the router away from the microwave.',
  'Landlord requires 48-hour notice before any site access.',
  'Primary contact is only reachable after 5 PM on weekdays.',
  'Verified backup contact before dispatching a technician.',
  'Customer asked about upgrading service; flagged for the sales team to follow up.',
  'Gate code required for site access; confirmed with customer during last visit.',
  'Customer prefers text message over phone calls for service updates.',
  'Noted a dog on the property; technician should call ahead before arrival.',
]

function buildNotes(customerId: string, installDate: Date): CustomerNote[] {
  const count = pickWeighted([[0, 5], [1, 4], [2, 3], [3, 1]] as const)
  const notes: CustomerNote[] = []
  for (let i = 0; i < count; i += 1) {
    const date = addDays(installDate, randomInt(0, Math.max(1, daysBetween(installDate, NOW))))
    notes.push({
      id: `${customerId}-NOTE-${i + 1}`,
      author: residentialName(),
      timestamp: formatAbsoluteDate(date),
      body: pick(NOTE_TEMPLATES),
    })
  }
  return notes.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1))
}

function buildCustomer(index: number): Customer {
  const type = pickWeighted(CUSTOMER_TYPE_WEIGHTS)
  const town = pick(TOWNS)
  const status = pickWeighted(STATUS_WEIGHTS)
  const installDate = status === 'pending' ? recentInstallDate() : randomInstallDate()

  const id = `CUST-${100000 + index}`
  const name = type === 'residential' ? residentialName() : businessName()
  const address = streetAddress()
  const postalCode = pick(town.postalCodes)
  const serviceAddress = `${address}, ${town.city}, ID ${postalCode}`
  const serviceStatus = serviceStatusFor(status)

  const primaryService = buildService(
    id,
    1,
    type,
    serviceStatus,
    installDate,
    serviceAddress,
    type === 'residential' ? RESIDENTIAL_SERVICES : BUSINESS_SERVICES,
  )

  const services = [primaryService]

  if (type === 'business' && status !== 'pending' && chance(0.2)) {
    const secondaryProvisioned = addDays(installDate, randomInt(30, 400))
    const secondaryStatus = chance(0.15) ? 'pending' : serviceStatus
    services.push(
      buildService(id, 2, type, secondaryStatus, secondaryProvisioned, serviceAddress, SECONDARY_BUSINESS_SERVICES),
    )
  }

  const emailDomain = type === 'business' ? `${slugify(name)}.example` : 'example.com'
  const { timeline, activity } = buildOperationalHistory(id, status, installDate, primaryService.tier)

  return {
    id,
    name,
    type,
    status,
    address,
    city: town.city,
    state: 'ID',
    postalCode,
    installDate: formatIsoDate(installDate),
    services,
    contacts: buildContacts(type, name, emailDomain),
    alerts: buildAlerts(id, status),
    activity,
    timeline,
    notes: buildNotes(id, installDate),
  }
}

const CUSTOMER_COUNT = 92

export const CUSTOMERS: Customer[] = Array.from({ length: CUSTOMER_COUNT }, (_, index) => buildCustomer(index))
