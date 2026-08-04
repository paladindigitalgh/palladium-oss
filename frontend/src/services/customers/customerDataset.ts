import type { Customer, CustomerService, CustomerStatus, CustomerType, ServiceTechnology } from '@/types/customer'

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

const RESIDENTIAL_SERVICES: [CustomerService, number][] = [
  [{ tier: '100 Mbps Residential Fiber', technology: 'gpon' }, 5],
  [{ tier: '250 Mbps Residential Fiber', technology: 'gpon' }, 6],
  [{ tier: '500 Mbps Residential Fiber', technology: 'gpon' }, 4],
  [{ tier: '1 Gbps Residential Fiber', technology: 'gpon' }, 3],
  [{ tier: '2 Gbps Residential Fiber', technology: 'xgs-pon' }, 2],
  [{ tier: '5 Gbps Residential Fiber', technology: 'xgs-pon' }, 1],
]

const BUSINESS_SERVICES: [CustomerService, number][] = [
  [{ tier: '50 Mbps Business Fiber', technology: 'gpon' }, 3],
  [{ tier: '250 Mbps Business Fiber', technology: 'gpon' }, 4],
  [{ tier: '1 Gbps Business Fiber', technology: 'xgs-pon' }, 4],
  [{ tier: '5 Gbps Business Fiber', technology: 'xgs-pon' }, 2],
  [{ tier: '10 Gbps Business Fiber', technology: 'xgs-pon' }, 1],
]

const GPON_EQUIPMENT = ['ONT Model GN-100', 'ONT Model GN-110', 'ONT Model GN-120']
const XGS_EQUIPMENT = ['ONT Model XG-400', 'ONT Model XG-410', 'ONT Model XG-420']

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

function randomInstallDate(): string {
  // Between 2016-01-01 and 2026-06-01, before this milestone's "today".
  const start = new Date(2016, 0, 1).getTime()
  const end = new Date(2026, 5, 1).getTime()
  const timestamp = start + random() * (end - start)
  return new Date(timestamp).toISOString().slice(0, 10)
}

function serviceForType(type: CustomerType): CustomerService {
  return pickWeighted(type === 'residential' ? RESIDENTIAL_SERVICES : BUSINESS_SERVICES)
}

function equipmentFor(technology: ServiceTechnology): string {
  return pick(technology === 'gpon' ? GPON_EQUIPMENT : XGS_EQUIPMENT)
}

function residentialName(): string {
  const first = random() < 0.5 ? pick(MALE_FIRST_NAMES) : pick(FEMALE_FIRST_NAMES)
  return `${first} ${pick(LAST_NAMES)}`
}

function businessName(): string {
  return `${pick(BUSINESS_PREFIXES)} ${pick(BUSINESS_SUFFIXES)}`
}

function streetAddress(): string {
  return `${randomInt(100, 9899)} ${pick(STREET_NAMES)} ${pick(STREET_SUFFIXES)}`
}

function buildCustomer(index: number): Customer {
  const type = pickWeighted(CUSTOMER_TYPE_WEIGHTS)
  const town = pick(TOWNS)
  const service = serviceForType(type)

  return {
    id: `CUST-${100000 + index}`,
    name: type === 'residential' ? residentialName() : businessName(),
    type,
    status: pickWeighted(STATUS_WEIGHTS),
    address: streetAddress(),
    city: town.city,
    state: 'ID',
    postalCode: pick(town.postalCodes),
    primaryService: service,
    equipment: equipmentFor(service.technology),
    installDate: randomInstallDate(),
  }
}

const CUSTOMER_COUNT = 92

export const CUSTOMERS: Customer[] = Array.from({ length: CUSTOMER_COUNT }, (_, index) => buildCustomer(index))
