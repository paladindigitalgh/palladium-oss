import { it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import type { WorkflowInstance } from '@/types/workflowInstance'
import type { AccessInterface } from '@/types/accessInterface'

/**
 * Mirrors useCustomerCollection.test.ts's mocking shape, one module per
 * repository this composable fans out to. `load()` runs once at
 * creation (no filter state to watch, see useDashboard.ts's own doc
 * comment), so `settle()` only ever needs to be awaited once per test,
 * not once per state change.
 */
const {
  listCustomers,
  listServices,
  listDevices,
  listAllWorkflowInstances,
  listAccessNetworks,
  listOLTs,
  listPONPorts,
  listAccessInterfaces,
  listRecentEvents,
} = vi.hoisted(() => ({
  listCustomers: vi.fn(),
  listServices: vi.fn(),
  listDevices: vi.fn(),
  listAllWorkflowInstances: vi.fn(),
  listAccessNetworks: vi.fn(),
  listOLTs: vi.fn(),
  listPONPorts: vi.fn(),
  listAccessInterfaces: vi.fn(),
  listRecentEvents: vi.fn(),
}))

vi.mock('@/services/customers/customerRepository', () => ({ listCustomers }))
vi.mock('@/services/services/serviceRepository', () => ({ listServices }))
vi.mock('@/services/devices/deviceRepository', () => ({ listDevices }))
vi.mock('@/services/workflow/workflowRepository', () => ({ listAllWorkflowInstances }))
vi.mock('@/services/accessNetworks/accessNetworkRepository', () => ({ listAccessNetworks }))
vi.mock('@/services/olts/oltRepository', () => ({ listOLTs }))
vi.mock('@/services/ponPorts/ponPortRepository', () => ({ listPONPorts }))
vi.mock('@/services/accessInterfaces/accessInterfaceRepository', () => ({ listAccessInterfaces }))
vi.mock('@/services/events/eventRepository', () => ({ listRecentEvents }))

import { useDashboard } from './useDashboard'

async function settle() {
  await nextTick()
  await nextTick()
}

function workflowInstance(overrides: Partial<WorkflowInstance> = {}): WorkflowInstance {
  return {
    id: 'w1',
    definitionName: 'provision-service',
    serviceId: 's1',
    requestedByUserId: null,
    status: 'Pending',
    retryCount: 0,
    errorMessage: null,
    startedAt: null,
    completedAt: null,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function accessInterface(overrides: Partial<AccessInterface> = {}): AccessInterface {
  return {
    id: 'ai1',
    ponPortId: 'p1',
    technology: 'GPON',
    name: 'AI-1',
    status: 'Active',
    description: '',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

function stubHappyPath() {
  listCustomers.mockResolvedValue({ items: [], total: 42 })
  listServices.mockResolvedValue({ items: [], total: 7 })
  listDevices.mockResolvedValue({ items: [], total: 13 })
  listAllWorkflowInstances.mockResolvedValue([
    workflowInstance({ id: 'w1', status: 'Pending' }),
    workflowInstance({ id: 'w2', status: 'Failed' }),
    workflowInstance({ id: 'w3', status: 'Succeeded' }),
    workflowInstance({ id: 'w4', status: 'Running' }),
    workflowInstance({ id: 'w5', status: 'Cancelled' }),
  ])
  listAccessNetworks.mockResolvedValue({ items: [], total: 3 })
  listOLTs.mockResolvedValue([{ id: 'olt1' }, { id: 'olt2' }])
  listPONPorts.mockResolvedValue([{ id: 'p1' }, { id: 'p2' }, { id: 'p3' }])
  listAccessInterfaces.mockResolvedValue([
    accessInterface({ id: 'ai1', status: 'Active' }),
    accessInterface({ id: 'ai2', status: 'Active' }),
    accessInterface({ id: 'ai3', status: 'Disabled' }),
  ])
  listRecentEvents.mockResolvedValue([{ id: 'e1' }])
}

beforeEach(() => {
  for (const fn of [
    listCustomers,
    listServices,
    listDevices,
    listAllWorkflowInstances,
    listAccessNetworks,
    listOLTs,
    listPONPorts,
    listAccessInterfaces,
    listRecentEvents,
  ]) {
    fn.mockReset()
  }
})

it('fetches every source in parallel on creation', async () => {
  stubHappyPath()

  useDashboard()
  await settle()

  expect(listCustomers).toHaveBeenCalledWith({ pageSize: 1 })
  expect(listServices).toHaveBeenCalledWith({ status: 'Active', pageSize: 1 })
  expect(listDevices).toHaveBeenCalledWith({ pageSize: 1 })
  expect(listAllWorkflowInstances).toHaveBeenCalled()
  expect(listAccessNetworks).toHaveBeenCalledWith({ pageSize: 1 })
  expect(listOLTs).toHaveBeenCalled()
  expect(listPONPorts).toHaveBeenCalled()
  expect(listAccessInterfaces).toHaveBeenCalled()
  expect(listRecentEvents).toHaveBeenCalledWith(10)
})

it('derives stats from the fetched totals and pending/failed workflow instances', async () => {
  stubHappyPath()

  const dashboard = useDashboard()
  await settle()

  expect(dashboard.stats.value).toEqual({ customers: 42, activeServices: 7, devices: 13, pendingTasks: 2 })
  expect(dashboard.pendingTasks.value.map((t) => t.id)).toEqual(['w1', 'w2'])
})

it('derives network overview counts, splitting access interfaces by status', async () => {
  stubHappyPath()

  const dashboard = useDashboard()
  await settle()

  expect(dashboard.networkOverview.value).toEqual({
    accessNetworks: 3,
    olts: 2,
    ponPorts: 3,
    activeInterfaces: 2,
    disabledInterfaces: 1,
  })
})

it('exposes recent events as recentActivity', async () => {
  stubHappyPath()

  const dashboard = useDashboard()
  await settle()

  expect(dashboard.recentActivity.value).toEqual([{ id: 'e1' }])
})

it('sets error and stops loading when any fetch rejects, without touching stale data', async () => {
  stubHappyPath()
  listDevices.mockRejectedValue(new Error('network down'))

  const dashboard = useDashboard()
  await settle()

  expect(dashboard.error.value).toBe(true)
  expect(dashboard.loading.value).toBe(false)
  expect(dashboard.stats.value).toEqual({ customers: 0, activeServices: 0, devices: 0, pendingTasks: 0 })
})

it('retry re-runs every fetch', async () => {
  stubHappyPath()
  const dashboard = useDashboard()
  await settle()
  listCustomers.mockClear()

  await dashboard.retry()

  expect(listCustomers).toHaveBeenCalledTimes(1)
})
