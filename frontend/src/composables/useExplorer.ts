import { computed, ref, watch } from 'vue'
import { listCustomersWithContacts, listDevicesReport, listCustomersWithDevices } from '@/services/reports/reportRepository'
import { rowsToCsv, downloadCsv } from '@/lib/csv'
import type { DataTableColumn } from '@/components/data-display/DataTable.vue'
import type { CustomerContactRow, DeviceReportRow, CustomerDeviceRow } from '@/types/report'

/**
 * Owns state and orchestration for the Explorer Workspace
 * (docs/09-WORKSPACE-SPECIFICATIONS.md §15) -- ExplorerView.vue is a
 * thin template over this, the same "composable owns logic, view owns
 * markup" split useDeviceCollection.ts already establishes for
 * DeviceCollectionView.vue.
 *
 * Deliberately not shaped like useDeviceCollection.ts: that composable
 * refetches from the network on every search/sort/page change. A report
 * here is fetched once, in full, when selected -- reports are pull-once
 * CSV-prep tools, not a live filtered browse list -- and every
 * reshaping after that (search, sort, pagination, CSV export) happens
 * entirely client-side against that one cached array.
 *
 * Each report is described by one concrete ReportDefinition (not a
 * generic schema, per this workspace's "curated reports, not a dynamic
 * query builder" decision): `toRecord` flattens that report's own typed
 * row into the one `Record<string, string>` shape the table, the search
 * filter, and rowsToCsv all share -- a report's id fields (e.g.
 * customerId) are included in that record even when not one of
 * `columns`, purely so `openRoute` can read them back out, and `key` is
 * that record's own unique identity for DataTable's `row-key`.
 */
export interface ReportRecord {
  key: string
  [field: string]: string
}

interface ReportDefinition {
  id: string
  label: string
  columns: DataTableColumn[]
  fetch: () => Promise<ReportRecord[]>
  openRoute: (row: ReportRecord) => string
  filename: string
}

function toReportRecord(key: string, fields: Record<string, string>): ReportRecord {
  return { key, ...fields }
}

const REPORTS: ReportDefinition[] = [
  {
    id: 'customers-contacts',
    label: 'Customers & Contacts',
    columns: [
      { key: 'customerName', label: 'Customer', sortable: true },
      { key: 'customerType', label: 'Type', sortable: true },
      { key: 'customerStatus', label: 'Status', sortable: true },
      { key: 'contactName', label: 'Contact Name', sortable: true },
      { key: 'contactRole', label: 'Role', sortable: true },
      { key: 'contactEmail', label: 'Email', sortable: true },
      { key: 'contactPhone', label: 'Phone', sortable: true },
      { key: 'contactStatus', label: 'Contact Status', sortable: true },
    ],
    async fetch() {
      const rows = await listCustomersWithContacts()
      return rows.map((row: CustomerContactRow) =>
        toReportRecord(`${row.customerId}-${row.contactId ?? 'none'}`, {
          customerId: row.customerId,
          customerName: row.customerName,
          customerType: row.customerType,
          customerStatus: row.customerStatus,
          contactName: row.contactName,
          contactRole: row.contactRole,
          contactEmail: row.contactEmail,
          contactPhone: row.contactPhone,
          contactStatus: row.contactStatus,
        }),
      )
    },
    openRoute: (row) => `/customers/${row.customerId}`,
    filename: 'customers-contacts',
  },
  {
    id: 'devices',
    label: 'Devices',
    columns: [
      { key: 'deviceName', label: 'Device', sortable: true },
      { key: 'manufacturer', label: 'Manufacturer', sortable: true },
      { key: 'model', label: 'Model', sortable: true },
      { key: 'serialNumber', label: 'Serial Number', sortable: true },
      { key: 'deviceStatus', label: 'Status', sortable: true },
      { key: 'siteName', label: 'Site', sortable: true },
      { key: 'buildingName', label: 'Building', sortable: true },
      { key: 'roomName', label: 'Room', sortable: true },
      { key: 'rackName', label: 'Rack', sortable: true },
      { key: 'assignedCustomerName', label: 'Assigned Customer', sortable: true },
    ],
    async fetch() {
      const rows = await listDevicesReport()
      return rows.map((row: DeviceReportRow) =>
        toReportRecord(row.deviceId, {
          deviceId: row.deviceId,
          deviceName: row.deviceName,
          manufacturer: row.manufacturer,
          model: row.model,
          serialNumber: row.serialNumber,
          assetTag: row.assetTag,
          deviceStatus: row.deviceStatus,
          siteName: row.siteName,
          buildingName: row.buildingName,
          roomName: row.roomName,
          rackName: row.rackName,
          assignedCustomerId: row.assignedCustomerId ?? '',
          assignedCustomerName: row.assignedCustomerName,
        }),
      )
    },
    openRoute: (row) => `/devices/${row.deviceId}`,
    filename: 'devices',
  },
  {
    id: 'customers-devices',
    label: 'Customers & Devices',
    columns: [
      { key: 'customerName', label: 'Customer', sortable: true },
      { key: 'customerType', label: 'Type', sortable: true },
      { key: 'customerStatus', label: 'Customer Status', sortable: true },
      { key: 'deviceName', label: 'Device', sortable: true },
      { key: 'serialNumber', label: 'Serial Number', sortable: true },
      { key: 'manufacturer', label: 'Manufacturer', sortable: true },
      { key: 'model', label: 'Model', sortable: true },
      { key: 'deviceStatus', label: 'Device Status', sortable: true },
      { key: 'relationship', label: 'Relationship', sortable: true },
      { key: 'serviceStatus', label: 'Service Status', sortable: true },
      { key: 'locationName', label: 'Location', sortable: true },
    ],
    async fetch() {
      const rows = await listCustomersWithDevices()
      return rows.map((row: CustomerDeviceRow, index: number) =>
        toReportRecord(`${row.customerId}-${row.deviceId}-${row.relationship}-${index}`, {
          customerId: row.customerId,
          customerName: row.customerName,
          customerType: row.customerType,
          customerStatus: row.customerStatus,
          deviceName: row.deviceName,
          serialNumber: row.serialNumber,
          manufacturer: row.manufacturer,
          model: row.model,
          deviceStatus: row.deviceStatus,
          relationship: row.relationship,
          serviceStatus: row.serviceStatus,
          locationName: row.locationName,
        }),
      )
    },
    openRoute: (row) => `/customers/${row.customerId}`,
    filename: 'customers-devices',
  },
]

const PAGE_SIZE = 25

export function useExplorer() {
  const selectedReportId = ref(REPORTS[0].id)
  const selectedReport = computed(() => REPORTS.find((r) => r.id === selectedReportId.value) ?? REPORTS[0])
  const reportOptions = REPORTS.map((r) => ({ value: r.id, label: r.label }))

  const search = ref('')
  const sortKey = ref('')
  const sortDirection = ref<'asc' | 'desc'>('asc')
  const page = ref(1)

  const allRows = ref<ReportRecord[]>([])
  const loading = ref(false)

  async function loadSelectedReport() {
    loading.value = true
    search.value = ''
    sortKey.value = selectedReport.value.columns[0]?.key ?? ''
    sortDirection.value = 'asc'
    page.value = 1
    try {
      allRows.value = await selectedReport.value.fetch()
    } finally {
      loading.value = false
    }
  }

  watch(selectedReportId, loadSelectedReport, { immediate: true })

  const filteredRows = computed(() => {
    const term = search.value.trim().toLowerCase()
    if (!term) return allRows.value
    return allRows.value.filter((row) => Object.values(row).some((value) => value.toLowerCase().includes(term)))
  })

  const sortedRows = computed(() => {
    const key = sortKey.value
    if (!key) return filteredRows.value
    const direction = sortDirection.value === 'desc' ? -1 : 1
    return filteredRows.value
      .slice()
      .sort((a, b) => (a[key] ?? '').localeCompare(b[key] ?? '') * direction)
  })

  const pageRows = computed(() => {
    const start = (page.value - 1) * PAGE_SIZE
    return sortedRows.value.slice(start, start + PAGE_SIZE)
  })

  function toggleSort(key: string) {
    if (sortKey.value === key) {
      sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortKey.value = key
      sortDirection.value = 'asc'
    }
  }

  watch(search, () => {
    page.value = 1
  })

  function openRoute(row: ReportRecord): string {
    return selectedReport.value.openRoute(row)
  }

  /** Exports every row currently matching `search` (not just the visible page) as a CSV download. */
  function exportCsv() {
    const csv = rowsToCsv(selectedReport.value.columns, sortedRows.value)
    const date = new Date().toISOString().slice(0, 10)
    downloadCsv(`${selectedReport.value.filename}-${date}.csv`, csv)
  }

  return {
    reportOptions,
    selectedReportId,
    selectedReport,
    search,
    sortKey,
    sortDirection,
    toggleSort,
    page,
    pageSize: PAGE_SIZE,
    pageRows,
    total: computed(() => sortedRows.value.length),
    loading,
    openRoute,
    exportCsv,
  }
}
