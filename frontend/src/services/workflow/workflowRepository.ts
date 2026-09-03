import type { WorkflowDefinitionName, WorkflowInstance } from '@/types/workflowInstance'
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

export async function listWorkflowInstancesByServiceId(serviceId: string): Promise<WorkflowInstance[]> {
  const { workflow_instances: instances } = await apiFetch<{ workflow_instances: WorkflowInstanceDto[] }>(
    `/workflow-instances/?service_id=${encodeURIComponent(serviceId)}`,
  )
  return instances.map(fromDto).sort((a, b) => b.createdAt.localeCompare(a.createdAt))
}

/**
 * Creates a WorkflowInstance and immediately executes it
 * (docs/05-WORKFLOW-ENGINE.md) -- this is the real mechanism behind a
 * Service Workspace's Provision/Suspend/Resume actions. Execution is
 * synchronous: the returned instance already reflects Succeeded or
 * Failed, there is no separate polling step.
 */
export async function runWorkflow(serviceId: string, definitionName: WorkflowDefinitionName): Promise<WorkflowInstance> {
  const created = await apiFetch<WorkflowInstanceDto>('/workflow-instances/', {
    method: 'POST',
    body: { service_id: serviceId, definition_name: definitionName },
  })

  const executed = await apiFetch<WorkflowInstanceDto>(`/workflow-instances/${created.id}/execute`, {
    method: 'POST',
  })

  return fromDto(executed)
}
