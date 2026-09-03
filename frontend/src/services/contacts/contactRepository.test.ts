import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listContacts, listContactsByCustomerId, getContactById, createContact, updateContact, deleteContact } from './contactRepository'

/**
 * Mirrors locationRepository.test.ts's shape exactly -- no client-side
 * search/sort/pagination, just slices of one fetched list.
 * getContactById filters the full list and returns null when absent; it
 * does not go through apiFetch's own not_found catch (there is no
 * per-id GET endpoint used here), so there is no ApiError branch to test
 * for it.
 */
const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/services/api/httpClient')>()
  return { ...actual, apiFetch }
})

function contactDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'ct1',
    customer_id: 'c1',
    name: 'Jane Doe',
    role: 'Primary',
    email: 'jane@example.com',
    phone: '555-0100',
    status: 'Active',
    description: '',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listContacts', () => {
  it('maps every contact from the DTO', async () => {
    apiFetch.mockResolvedValue({ contacts: [contactDto({ id: 'ct1' }), contactDto({ id: 'ct2' })] })

    const result = await listContacts()

    expect(result.map((c) => c.id)).toEqual(['ct1', 'ct2'])
  })
})

describe('listContactsByCustomerId', () => {
  it('returns only contacts belonging to the given customer', async () => {
    apiFetch.mockResolvedValue({
      contacts: [
        contactDto({ id: 'ct1', customer_id: 'c1' }),
        contactDto({ id: 'ct2', customer_id: 'c2' }),
        contactDto({ id: 'ct3', customer_id: 'c1' }),
      ],
    })

    const result = await listContactsByCustomerId('c1')

    expect(result.map((c) => c.id)).toEqual(['ct1', 'ct3'])
  })
})

describe('getContactById', () => {
  it('returns the contact when found', async () => {
    apiFetch.mockResolvedValue({ contacts: [contactDto({ id: 'ct1' }), contactDto({ id: 'ct2' })] })

    const result = await getContactById('ct2')

    expect(result?.id).toBe('ct2')
  })

  it('returns null instead of throwing when the contact does not exist', async () => {
    apiFetch.mockResolvedValue({ contacts: [contactDto({ id: 'ct1' })] })

    const result = await getContactById('missing')

    expect(result).toBeNull()
  })
})

describe('createContact', () => {
  it('sends the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(contactDto({ id: 'new' }))

    await createContact({
      customerId: 'c1',
      name: 'John Smith',
      role: 'Billing',
      email: 'john@example.com',
      phone: '555-0199',
      status: 'Active',
      description: 'New billing contact',
    })

    expect(apiFetch).toHaveBeenCalledWith('/contacts/', {
      method: 'POST',
      body: {
        customer_id: 'c1',
        name: 'John Smith',
        role: 'Billing',
        email: 'john@example.com',
        phone: '555-0199',
        status: 'Active',
        description: 'New billing contact',
      },
    })
  })
})

describe('updateContact', () => {
  it('sends a PUT with the request body in the API wire shape', async () => {
    apiFetch.mockResolvedValue(contactDto({ id: 'ct1', name: 'Jane Renamed' }))

    await updateContact('ct1', {
      customerId: 'c1',
      name: 'Jane Renamed',
      role: 'Technical',
      email: 'jane.renamed@example.com',
      phone: '555-0111',
      status: 'Inactive',
      description: 'Updated',
    })

    expect(apiFetch).toHaveBeenCalledWith('/contacts/ct1', {
      method: 'PUT',
      body: {
        customer_id: 'c1',
        name: 'Jane Renamed',
        role: 'Technical',
        email: 'jane.renamed@example.com',
        phone: '555-0111',
        status: 'Inactive',
        description: 'Updated',
      },
    })
  })
})

describe('deleteContact', () => {
  it('issues a DELETE request for the given id', async () => {
    apiFetch.mockResolvedValue(undefined)

    await deleteContact('ct1')

    expect(apiFetch).toHaveBeenCalledWith('/contacts/ct1', { method: 'DELETE' })
  })
})
