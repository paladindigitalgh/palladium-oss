import { ref } from 'vue'
import { listCustomers } from '@/services/customers/customerRepository'
import { listServices } from '@/services/services/serviceRepository'
import { listDevices } from '@/services/devices/deviceRepository'
import { listAllWorkflowInstances } from '@/services/workflow/workflowRepository'
import { listAccessNetworks } from '@/services/accessNetworks/accessNetworkRepository'
import { listOLTs } from '@/services/olts/oltRepository'
import { listPONPorts } from '@/services/ponPorts/ponPortRepository'
import { listAccessInterfaces } from '@/services/accessInterfaces/accessInterfaceRepository'
import { listRecentEvents } from '@/services/events/eventRepository'
import type { WorkflowInstance } from '@/types/workflowInstance'
import type { TimelineEvent } from '@/types/timelineEvent'

export interface DashboardStats {
  customers: number
  activeServices: number
  devices: number
  pendingTasks: number
}

export interface DashboardNetworkOverview {
  accessNetworks: number
  olts: number
  ponPorts: number
  activeInterfaces: number
  disabledInterfaces: number
}

/**
 * A WorkflowInstance an operator needs to do something about: it either
 * hasn't run yet, or it ran and failed. Running/Succeeded/Cancelled all
 * need no attention -- Succeeded is done, Cancelled was a deliberate
 * choice, and Running is already in flight (this app's workflow
 * execution is synchronous, so in practice this only ever shows a
 * momentary state, but it is still not "needs attention").
 */
const PENDING_TASK_STATUSES = new Set<WorkflowInstance['status']>(['Pending', 'Failed'])

/**
 * Owns the Dashboard's data. Unlike every use*Collection.ts composable
 * (one paginated list, refetched on filter/page change), this fans out
 * across every domain the Dashboard summarizes in parallel and derives
 * counts from the results -- there is no filter state to watch, so
 * `load` runs once at creation rather than being driven by a `watch`.
 * Same `loading`/`error`/`retry` shape as the collection composables,
 * for the same reason: a failed fetch should leave the view able to
 * show an error state and retry, not throw.
 */
export function useDashboard() {
  const loading = ref(true)
  const error = ref(false)
  const stats = ref<DashboardStats>({ customers: 0, activeServices: 0, devices: 0, pendingTasks: 0 })
  const networkOverview = ref<DashboardNetworkOverview>({
    accessNetworks: 0,
    olts: 0,
    ponPorts: 0,
    activeInterfaces: 0,
    disabledInterfaces: 0,
  })
  const recentActivity = ref<TimelineEvent[]>([])
  const pendingTasks = ref<WorkflowInstance[]>([])

  async function load() {
    loading.value = true
    error.value = false

    try {
      const [customers, services, devices, workflowInstances, accessNetworks, olts, ponPorts, accessInterfaces, events] =
        await Promise.all([
          listCustomers({ pageSize: 1 }),
          listServices({ status: 'Active', pageSize: 1 }),
          listDevices({ pageSize: 1 }),
          listAllWorkflowInstances(),
          listAccessNetworks({ pageSize: 1 }),
          listOLTs(),
          listPONPorts(),
          listAccessInterfaces(),
          listRecentEvents(10),
        ])

      const pending = workflowInstances.filter((instance) => PENDING_TASK_STATUSES.has(instance.status))

      stats.value = {
        customers: customers.total,
        activeServices: services.total,
        devices: devices.total,
        pendingTasks: pending.length,
      }
      networkOverview.value = {
        accessNetworks: accessNetworks.total,
        olts: olts.length,
        ponPorts: ponPorts.length,
        activeInterfaces: accessInterfaces.filter((iface) => iface.status === 'Active').length,
        disabledInterfaces: accessInterfaces.filter((iface) => iface.status === 'Disabled').length,
      }
      recentActivity.value = events
      pendingTasks.value = pending
    } catch {
      error.value = true
    } finally {
      loading.value = false
    }
  }

  load()

  return { loading, error, stats, networkOverview, recentActivity, pendingTasks, retry: load }
}
