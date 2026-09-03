/**
 * The Workflow Instance domain type (internal/workflow), matching
 * internal/workflow/httpapi/dto.go's instanceResponse. This is what a
 * Service Workspace's Provision/Suspend/Resume actions create and
 * execute (docs/05-WORKFLOW-ENGINE.md) -- the real mechanism behind
 * those buttons, not a disabled placeholder.
 */
export type WorkflowDefinitionName =
  | 'provision-service'
  | 'reprovision-service'
  | 'suspend-service'
  | 'resume-service'
  | 'disconnect-service'
  | 'synchronize-service'

export type WorkflowStatus = 'Pending' | 'Running' | 'Succeeded' | 'Failed' | 'Cancelled'

export interface WorkflowInstance {
  id: string
  definitionName: WorkflowDefinitionName
  serviceId: string
  requestedByUserId: string | null
  status: WorkflowStatus
  retryCount: number
  errorMessage: string | null
  startedAt: string | null
  completedAt: string | null
  createdAt: string
  updatedAt: string
}
