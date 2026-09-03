import type { Contact } from '@/types/contact'
import { apiFetch } from '@/services/api/httpClient'

/**
 * The real Contact data source. GET /contacts has no server-side
 * filtering (see internal/contact/httpapi), so "a customer's Contacts"
 * is resolved by fetching every Contact once and filtering client-side --
 * the same pattern locationRepository.ts uses.
 */

interface ContactDto {
  id: string
  customer_id: string
  name: string
  role: Contact['role']
  email: string
  phone: string
  status: Contact['status']
  description: string
}

function fromDto(dto: ContactDto): Contact {
  return {
    id: dto.id,
    customerId: dto.customer_id,
    name: dto.name,
    role: dto.role,
    email: dto.email,
    phone: dto.phone,
    status: dto.status,
    description: dto.description,
  }
}

export async function listContacts(): Promise<Contact[]> {
  const { contacts } = await apiFetch<{ contacts: ContactDto[] }>('/contacts/')
  return contacts.map(fromDto)
}

export async function listContactsByCustomerId(customerId: string): Promise<Contact[]> {
  const contacts = await listContacts()
  return contacts.filter((contact) => contact.customerId === customerId)
}

export async function getContactById(id: string): Promise<Contact | null> {
  const contacts = await listContacts()
  return contacts.find((contact) => contact.id === id) ?? null
}

export interface CreateContactInput {
  customerId: string
  name: string
  role: Contact['role']
  email: string
  phone: string
  status: Contact['status']
  description: string
}

export async function createContact(input: CreateContactInput): Promise<Contact> {
  const dto = await apiFetch<ContactDto>('/contacts/', {
    method: 'POST',
    body: {
      customer_id: input.customerId,
      name: input.name,
      role: input.role,
      email: input.email,
      phone: input.phone,
      status: input.status,
      description: input.description,
    },
  })
  return fromDto(dto)
}

export interface UpdateContactInput {
  customerId: string
  name: string
  role: Contact['role']
  email: string
  phone: string
  status: Contact['status']
  description: string
}

export async function updateContact(id: string, input: UpdateContactInput): Promise<Contact> {
  const dto = await apiFetch<ContactDto>(`/contacts/${id}`, {
    method: 'PUT',
    body: {
      customer_id: input.customerId,
      name: input.name,
      role: input.role,
      email: input.email,
      phone: input.phone,
      status: input.status,
      description: input.description,
    },
  })
  return fromDto(dto)
}

/**
 * Deletes the Contact identified by id. Unlike deleteLocation/
 * deleteCustomer, this never throws an ApiError with kind "conflict":
 * contacts.customer_id is ON DELETE CASCADE, not RESTRICT (see
 * internal/contact/postgres/contact.go's package doc comment), and
 * nothing else references a Contact, so there is no foreign key
 * relationship a delete could violate in either direction.
 */
export async function deleteContact(id: string): Promise<void> {
  await apiFetch<void>(`/contacts/${id}`, { method: 'DELETE' })
}
