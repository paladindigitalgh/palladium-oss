import { describe, it, expect, vi, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useExplorerReports } from './useExplorerReports'

const { listCustomersWithContacts, listDevicesReport, listCustomersWithDevices } = vi.hoisted(() => ({
  listCustomersWithContacts: vi.fn(),
  listDevicesReport: vi.fn(),
  listCustomersWithDevices: vi.fn(),
}))

vi.mock('@/services/reports/reportRepository', () => ({
  listCustomersWithContacts,
  listDevicesReport,
  listCustomersWithDevices,
}))

const { rowsToCsv, downloadCsv } = vi.hoisted(() => ({
  rowsToCsv: vi.fn((..._args: unknown[]) => 'csv-content'),
  downloadCsv: vi.fn((..._args: unknown[]) => undefined),
}))

vi.mock('@/lib/csv', () => ({ rowsToCsv, downloadCsv }))

async function settle() {
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  listCustomersWithContacts.mockReset().mockResolvedValue([])
  listDevicesReport.mockReset().mockResolvedValue([])
  listCustomersWithDevices.mockReset().mockResolvedValue([])
  rowsToCsv.mockClear()
  downloadCsv.mockClear()
})

it('fetches nothing on creation -- no report is selected until the user picks one', async () => {
  const explorer = useExplorerReports()
  await settle()

  expect(listCustomersWithContacts).not.toHaveBeenCalled()
  expect(listDevicesReport).not.toHaveBeenCalled()
  expect(listCustomersWithDevices).not.toHaveBeenCalled()
  expect(explorer.selectedReport.value).toBeNull()
})

it('exposes all three reports for the tile picker', () => {
  const explorer = useExplorerReports()

  expect(explorer.reports.map((r) => r.id)).toEqual(['customers-contacts', 'devices', 'customers-devices'])
  expect(explorer.reports.every((r) => r.label && r.description)).toBe(true)
})

it('selectReport fetches the chosen report', async () => {
  const explorer = useExplorerReports()

  explorer.selectReport('customers-contacts')
  await settle()

  expect(listCustomersWithContacts).toHaveBeenCalledTimes(1)
  expect(listDevicesReport).not.toHaveBeenCalled()
  expect(listCustomersWithDevices).not.toHaveBeenCalled()
})

it('flattens a Customers & Contacts row into the table/CSV shape', async () => {
  listCustomersWithContacts.mockResolvedValue([
    {
      customerId: 'c1',
      customerName: 'Acme',
      customerType: 'Business',
      customerStatus: 'Active',
      contactId: 'ct1',
      contactName: 'Jane Doe',
      contactRole: 'Primary',
      contactEmail: 'jane@example.com',
      contactPhone: '555-0100',
      contactStatus: 'Active',
    },
  ])

  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()

  expect(explorer.pageRows.value).toEqual([
    expect.objectContaining({
      key: 'c1-ct1',
      customerId: 'c1',
      customerName: 'Acme',
      contactName: 'Jane Doe',
      contactEmail: 'jane@example.com',
    }),
  ])
  expect(explorer.total.value).toBe(1)
})

it('switching reports fetches only the newly selected report', async () => {
  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()
  listCustomersWithContacts.mockClear()

  explorer.selectReport('devices')
  await settle()

  expect(listDevicesReport).toHaveBeenCalledTimes(1)
  expect(listCustomersWithContacts).not.toHaveBeenCalled()
})

it('switching reports resets search and page', async () => {
  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()

  explorer.search.value = 'acme'
  explorer.page.value = 2
  await settle()

  explorer.selectReport('devices')
  await settle()

  expect(explorer.search.value).toBe('')
  expect(explorer.page.value).toBe(1)
})

it('search filters rows client-side without refetching', async () => {
  listCustomersWithContacts.mockResolvedValue([
    {
      customerId: 'c1',
      customerName: 'Acme',
      customerType: 'Business',
      customerStatus: 'Active',
      contactId: 'ct1',
      contactName: 'Jane Doe',
      contactRole: 'Primary',
      contactEmail: 'jane@example.com',
      contactPhone: '555-0100',
      contactStatus: 'Active',
    },
    {
      customerId: 'c2',
      customerName: 'Widgets Inc',
      customerType: 'Business',
      customerStatus: 'Active',
      contactId: 'ct2',
      contactName: 'John Smith',
      contactRole: 'Primary',
      contactEmail: 'john@example.com',
      contactPhone: '555-0101',
      contactStatus: 'Active',
    },
  ])

  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()
  listCustomersWithContacts.mockClear()

  explorer.search.value = 'widgets'
  await settle()

  expect(listCustomersWithContacts).not.toHaveBeenCalled()
  expect(explorer.pageRows.value).toHaveLength(1)
  expect(explorer.pageRows.value[0].customerName).toBe('Widgets Inc')
})

it('search resets to page 1', async () => {
  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()

  explorer.page.value = 3
  await settle()

  explorer.search.value = 'x'
  await settle()

  expect(explorer.page.value).toBe(1)
})

describe('toggleSort', () => {
  it('flips direction when toggling the same key', async () => {
    const explorer = useExplorerReports()
    explorer.selectReport('customers-contacts')
    await settle()

    explorer.toggleSort('customerName')
    expect(explorer.sortDirection.value).toBe('desc')

    explorer.toggleSort('customerName')
    expect(explorer.sortDirection.value).toBe('asc')
  })

  it('switches key and resets to ascending on a different key', async () => {
    const explorer = useExplorerReports()
    explorer.selectReport('customers-contacts')
    await settle()

    explorer.toggleSort('customerType')

    expect(explorer.sortKey.value).toBe('customerType')
    expect(explorer.sortDirection.value).toBe('asc')
  })
})

it('openRoute for a Customers & Contacts row points at the Customer detail route', async () => {
  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()

  expect(explorer.openRoute({ key: 'c1-ct1', customerId: 'c1' })).toBe('/customers/c1')
})

it('openRoute for a Devices row points at the Device detail route', async () => {
  const explorer = useExplorerReports()
  explorer.selectReport('devices')
  await settle()

  expect(explorer.openRoute({ key: 'd1', deviceId: 'd1' })).toBe('/devices/d1')
})

it('exportCsv passes the search-filtered rows (not just the current page) to rowsToCsv, and triggers a download', async () => {
  listCustomersWithContacts.mockResolvedValue([
    {
      customerId: 'c1',
      customerName: 'Acme',
      customerType: 'Business',
      customerStatus: 'Active',
      contactId: 'ct1',
      contactName: 'Jane',
      contactRole: 'Primary',
      contactEmail: 'jane@example.com',
      contactPhone: '555-0100',
      contactStatus: 'Active',
    },
    {
      customerId: 'c2',
      customerName: 'Widgets Inc',
      customerType: 'Business',
      customerStatus: 'Active',
      contactId: 'ct2',
      contactName: 'John',
      contactRole: 'Primary',
      contactEmail: 'john@example.com',
      contactPhone: '555-0101',
      contactStatus: 'Active',
    },
  ])

  const explorer = useExplorerReports()
  explorer.selectReport('customers-contacts')
  await settle()

  explorer.search.value = 'acme'
  await settle()

  explorer.exportCsv()

  expect(rowsToCsv).toHaveBeenCalledWith(
    explorer.selectedReport.value?.columns,
    expect.arrayContaining([expect.objectContaining({ customerName: 'Acme' })]),
  )
  const exportedRows = rowsToCsv.mock.calls[0][1]
  expect(exportedRows).toHaveLength(1)
  expect(downloadCsv).toHaveBeenCalledTimes(1)
  expect(downloadCsv.mock.calls[0][0]).toMatch(/^customers-contacts-\d{4}-\d{2}-\d{2}\.csv$/)
})

it('exportCsv does nothing when no report is selected', () => {
  const explorer = useExplorerReports()

  explorer.exportCsv()

  expect(rowsToCsv).not.toHaveBeenCalled()
  expect(downloadCsv).not.toHaveBeenCalled()
})
