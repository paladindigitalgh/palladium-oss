import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listCustomersWithContacts, listDevicesReport, listCustomersWithDevices } from './reportRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

const NIL_UUID = '00000000-0000-0000-0000-000000000000'

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listCustomersWithContacts', () => {
  it('maps a row with a real contact', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          customer_id: 'c1',
          customer_name: 'Acme',
          customer_type: 'Business',
          customer_status: 'Active',
          contact_id: 'ct1',
          contact_name: 'Jane Doe',
          contact_role: 'Primary',
          contact_email: 'jane@example.com',
          contact_phone: '555-0100',
          contact_status: 'Active',
        },
      ],
    })

    const result = await listCustomersWithContacts()

    expect(apiFetch).toHaveBeenCalledWith('/reports/customers-contacts')
    expect(result).toEqual([
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
  })

  it('maps the nil-UUID sentinel contact_id to null for a customer with no contact', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          customer_id: 'c1',
          customer_name: 'Acme',
          customer_type: 'Business',
          customer_status: 'Active',
          contact_id: NIL_UUID,
          contact_name: '',
          contact_role: '',
          contact_email: '',
          contact_phone: '',
          contact_status: '',
        },
      ],
    })

    const result = await listCustomersWithContacts()

    expect(result[0].contactId).toBeNull()
  })
})

describe('listDevicesReport', () => {
  it('maps a row with an assigned customer', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          device_id: 'd1',
          device_name: 'OLT-01',
          serial_number: 'SN1',
          asset_tag: 'AT1',
          device_status: 'Active',
          manufacturer: 'Nokia',
          model: 'G-010G',
          site_name: 'Main DC',
          building_name: 'Bldg A',
          room_name: 'Room 2',
          rack_name: 'Rack 3',
          assigned_customer_id: 'c1',
          assigned_customer_name: 'Acme',
        },
      ],
    })

    const result = await listDevicesReport()

    expect(apiFetch).toHaveBeenCalledWith('/reports/devices')
    expect(result[0].assignedCustomerId).toBe('c1')
    expect(result[0].assignedCustomerName).toBe('Acme')
    expect(result[0].siteName).toBe('Main DC')
  })

  it('maps the nil-UUID sentinel assigned_customer_id to null for an unassigned device', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          device_id: 'd1',
          device_name: 'OLT-01',
          serial_number: 'SN1',
          asset_tag: '',
          device_status: 'Unused',
          manufacturer: 'Nokia',
          model: 'G-010G',
          site_name: '',
          building_name: '',
          room_name: '',
          rack_name: '',
          assigned_customer_id: NIL_UUID,
          assigned_customer_name: '',
        },
      ],
    })

    const result = await listDevicesReport()

    expect(result[0].assignedCustomerId).toBeNull()
  })
})

describe('listCustomersWithDevices', () => {
  it('maps a Placement row', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          customer_id: 'c1',
          customer_name: 'Acme',
          customer_type: 'Business',
          customer_status: 'Active',
          device_id: 'd1',
          device_name: 'ONT-01',
          serial_number: 'SN1',
          manufacturer: 'Nokia',
          model: 'G-010G',
          device_status: 'Active',
          relationship: 'Placement',
          service_status: '',
          location_name: '',
        },
      ],
    })

    const result = await listCustomersWithDevices()

    expect(apiFetch).toHaveBeenCalledWith('/reports/customers-devices')
    expect(result[0].relationship).toBe('Placement')
    expect(result[0].locationName).toBe('')
  })

  it('maps a Service Equipment row', async () => {
    apiFetch.mockResolvedValue({
      rows: [
        {
          customer_id: 'c1',
          customer_name: 'Acme',
          customer_type: 'Business',
          customer_status: 'Active',
          device_id: 'd1',
          device_name: 'ONT-01',
          serial_number: 'SN1',
          manufacturer: 'Nokia',
          model: 'G-010G',
          device_status: 'Active',
          relationship: 'Service Equipment',
          service_status: 'Active',
          location_name: 'Main Office',
        },
      ],
    })

    const result = await listCustomersWithDevices()

    expect(result[0].relationship).toBe('Service Equipment')
    expect(result[0].serviceStatus).toBe('Active')
    expect(result[0].locationName).toBe('Main Office')
  })
})
