import type { ProjectStatus, TaskStatus, IssueStatus, ReworkStatus, MaterialReqStatus, RoutingStatus } from '@/types'

type Status = ProjectStatus | TaskStatus | IssueStatus | ReworkStatus | MaterialReqStatus | RoutingStatus | string

const MAP: Record<string, string> = {
  // Project
  CREATED: 'badge-gray', ROUTING: 'badge-blue', IN_PROGRESS: 'badge-purple',
  COMPLETED: 'badge-green', ARCHIVED: 'badge-gray',
  // Task
  PENDING: 'badge-gray', HOLD: 'badge-yellow', ISSUE_HOLD: 'badge-red',
  // Issue
  OPEN: 'badge-yellow', PENDING_APPROVAL: 'badge-blue', APPROVED: 'badge-green',
  REJECTED: 'badge-red', RESOLVED: 'badge-green', CLOSED: 'badge-gray',
  // Routing
  DRAFT: 'badge-gray', ACTIVE: 'badge-green', SUPERSEDED: 'badge-gray',
  // Material
  FULFILLED: 'badge-green',
}

const LABELS: Record<string, string> = {
  IN_PROGRESS: 'In Progress', ISSUE_HOLD: 'Issue Hold', PENDING_APPROVAL: 'Pending Review',
}

export default function StatusBadge({ status }: { status: Status }) {
  const cls = MAP[status] ?? 'badge-gray'
  const label = LABELS[status] ?? status.replace(/_/g, ' ')
  return <span className={cls}>{label}</span>
}
