import type { WorkflowDefinitionName, WorkflowInstance, WorkflowStatus } from '@/types/workflowInstance'
import { apiFetch } from '@/services/api/httpClient'

interface WorkflowInstanceDto {
  id: string
  definition_name: WorkflowDefinitionName
  service_id: string
  requested_by_user_id: string | null
  status: WorkflowInstance['status']
  retry_count: number
  error_message: string | null
  started_at: string | null
  completed_at: string | null
  created_at: string
  updated_at: string
}

function fromDto(dto: WorkflowInstanceDto): WorkflowInstance {
  return {
    id: dto.id,
    definitionName: dto.definition_name,
    serviceId: dto.service_id,
    requestedByUserId: dto.requested_by_user_id,
    status: dto.status,
    retryCount: dto.retry_count,
    errorMessage: dto.error_message,
    startedAt: dto.started_at,
    completedAt: dto.completed_at,
    createdAt: dto.created_at,
    updatedAt: dto.updated_at,
  }
}

/** Fetches every WorkflowInstance system-wide -- the Dashboard's Pending Tasks widget, not scoped to one Service. */
export async function listAllWorkflowInstances(): Promise<WorkflowInstance[]> {
  const { workflow_instances: instances } = await apiFetch<{ workflow_instances: WorkflowInstanceDto[] }>('/workflow-instances/')
  return instances.map(fromDto)
}

export async function listWorkflowInstancesByServiceId(serviceId: string): Promise<WorkflowInstance[]> {
  const { workflow_instances: instances } = await apiFetch<{ workflow_instances: WorkflowInstanceDto[] }>(
    `/workflow-instances/?service_id=${encodeURIComponent(serviceId)}`,
  )
  return instances.map(fromDto).sort((a, b) => b.createdAt.localeCompare(a.createdAt))
}

export async function getWorkflowInstance(id: string): Promise<WorkflowInstance> {
  const dto = await apiFetch<WorkflowInstanceDto>(`/workflow-instances/${id}`)
  return fromDto(dto)
}

const TERMINAL_STATUSES: WorkflowStatus[] = ['Succeeded', 'Failed', 'Cancelled']

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * Creates a WorkflowInstance and polls it until a terminal status
 * (docs/05-WORKFLOW-ENGINE.md) -- this is the real mechanism behind a
 * Service Workspace's Provision/Suspend/Resume actions. Execution is no
 * longer synchronous with creation: internal/workflow/worker.Worker
 * picks up the new Pending instance on its own poll cycle, so this
 * function polls GET .../{id} in the meantime rather than getting a
 * finished result back from a single execute call.
 *
 * pollIntervalMs/maxAttempts are overridable purely so tests don't have
 * to wait out real timers; callers outside the test suite should accept
 * the defaults. The default 1s/30 gives the worker up to 30s (its
 * default poll interval is 2s -- see WORKFLOW_WORKER_POLL_INTERVAL --
 * plus whatever the plugin call itself takes) before runWorkflow gives
 * up and reports the wait as taking too long; the instance itself keeps
 * running regardless, and a later page load will show its real outcome.
 */
export async function runWorkflow(
  serviceId: string,
  definitionName: WorkflowDefinitionName,
  pollIntervalMs = 1000,
  maxAttempts = 30,
): Promise<WorkflowInstance> {
  const created = await apiFetch<WorkflowInstanceDto>('/workflow-instances/', {
    method: 'POST',
    body: { service_id: serviceId, definition_name: definitionName },
  })

  let instance = fromDto(created)
  for (let attempt = 0; attempt < maxAttempts && !TERMINAL_STATUSES.includes(instance.status); attempt++) {
    await sleep(pollIntervalMs)
    instance = await getWorkflowInstance(instance.id)
  }

  if (!TERMINAL_STATUSES.includes(instance.status)) {
    throw new Error('This workflow is taking longer than expected -- check back shortly for its result.')
  }

  return instance
}
