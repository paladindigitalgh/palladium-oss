import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listAllWorkflowInstances, listWorkflowInstancesByServiceId, runWorkflow } from './workflowRepository'

const { apiFetch } = vi.hoisted(() => ({ apiFetch: vi.fn() }))

vi.mock('@/services/api/httpClient', () => ({ apiFetch }))

function instanceDto(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 'w1',
    definition_name: 'provision-service',
    service_id: 's1',
    requested_by_user_id: null,
    status: 'Succeeded',
    retry_count: 0,
    error_message: null,
    started_at: '2026-01-01T00:00:00Z',
    completed_at: '2026-01-01T00:01:00Z',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:01:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  apiFetch.mockReset()
})

describe('listWorkflowInstancesByServiceId', () => {
  it('builds the query string from the given service id', async () => {
    apiFetch.mockResolvedValue({ workflow_instances: [] })

    await listWorkflowInstancesByServiceId('s1')

    expect(apiFetch).toHaveBeenCalledWith('/workflow-instances/?service_id=s1')
  })

  it('sorts results by createdAt descending, most recent first', async () => {
    apiFetch.mockResolvedValue({
      workflow_instances: [
        instanceDto({ id: 'w1', created_at: '2026-01-01T00:00:00Z' }),
        instanceDto({ id: 'w2', created_at: '2026-01-03T00:00:00Z' }),
        instanceDto({ id: 'w3', created_at: '2026-01-02T00:00:00Z' }),
      ],
    })

    const result = await listWorkflowInstancesByServiceId('s1')

    expect(result.map((w) => w.id)).toEqual(['w2', 'w3', 'w1'])
  })
})

describe('listAllWorkflowInstances', () => {
  it('fetches every instance system-wide, with no query string', async () => {
    apiFetch.mockResolvedValue({
      workflow_instances: [instanceDto({ id: 'w1' }), instanceDto({ id: 'w2', service_id: 's2' })],
    })

    const result = await listAllWorkflowInstances()

    expect(apiFetch).toHaveBeenCalledWith('/workflow-instances/')
    expect(result.map((w) => w.id)).toEqual(['w1', 'w2'])
  })

  it('does not sort -- unlike listWorkflowInstancesByServiceId, callers see API order as-is', async () => {
    apiFetch.mockResolvedValue({
      workflow_instances: [
        instanceDto({ id: 'w1', created_at: '2026-01-01T00:00:00Z' }),
        instanceDto({ id: 'w2', created_at: '2026-01-03T00:00:00Z' }),
      ],
    })

    const result = await listAllWorkflowInstances()

    expect(result.map((w) => w.id)).toEqual(['w1', 'w2'])
  })
})

describe('runWorkflow', () => {
  it('creates a WorkflowInstance then immediately executes it, returning the executed result', async () => {
    const created = instanceDto({ id: 'w1', status: 'Pending' })
    const executed = instanceDto({ id: 'w1', status: 'Succeeded' })
    apiFetch.mockResolvedValueOnce(created).mockResolvedValueOnce(executed)

    const result = await runWorkflow('s1', 'provision-service')

    expect(apiFetch).toHaveBeenNthCalledWith(1, '/workflow-instances/', {
      method: 'POST',
      body: { service_id: 's1', definition_name: 'provision-service' },
    })
    expect(apiFetch).toHaveBeenNthCalledWith(2, '/workflow-instances/w1/execute', { method: 'POST' })
    expect(result.status).toBe('Succeeded')
  })
})
