import { useParams, useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { projectApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import { ArrowLeft, Clock, Circle } from 'lucide-react'
import { format } from 'date-fns'

const ACTION_COLOR: Record<string, string> = {
  PROJECT_CREATED: 'bg-green-500',
  PROJECT_UPDATED: 'bg-brand-500',
  PROJECT_STATUS_CHANGED: 'bg-purple-500',
  ROUTING_CREATED: 'bg-blue-500',
  TASK_STATUS_CHANGED: 'bg-yellow-500',
  SUBTASK_COMPLETED: 'bg-teal-500',
  ISSUE_RAISED: 'bg-red-500',
  ISSUE_RESOLVED: 'bg-green-500',
  REWORK_REQUESTED: 'bg-orange-500',
  REWORK_APPROVED: 'bg-blue-500',
  DAILY_REPORT_SUBMITTED: 'bg-gray-400',
  QUERY_CREATED: 'bg-indigo-500',
  MATERIAL_REQUESTED: 'bg-amber-500',
}

export default function ProjectTimelinePage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data: timelineRes, isLoading } = useQuery({
    queryKey: ['timeline', projectId],
    queryFn: () => projectApi.timeline(projectId!),
    refetchInterval: 30_000,
  })

  const logs = timelineRes?.data?.data ?? []
  const project = pRes?.data?.data

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-2xl space-y-5">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
        <div>
          <h1 className="page-title">Project Timeline</h1>
          {project && <p className="text-sm text-gray-500">{project.name} · {project.po_number}</p>}
        </div>
      </div>

      {logs.length === 0 ? (
        <EmptyState icon={Clock} title="No activity yet" description="Actions on this project will appear here." />
      ) : (
        <div className="relative">
          {/* vertical line */}
          <div className="absolute left-3.5 top-0 bottom-0 w-0.5 bg-gray-200" />
          <div className="space-y-1">
            {logs.map((log) => (
              <div key={log.id} className="relative flex gap-4 pl-10">
                <div className={`absolute left-1.5 top-2 h-4 w-4 rounded-full flex items-center justify-center ${ACTION_COLOR[log.action] ?? 'bg-gray-400'}`}>
                  <Circle size={6} className="text-white fill-white" />
                </div>
                <div className="flex-1 card card-body py-3 px-4 mb-2">
                  <div className="flex items-start justify-between gap-2">
                    <div>
                      <p className="text-sm font-medium text-gray-800">
                        {log.action.replace(/_/g, ' ').toLowerCase().replace(/^\w/, c => c.toUpperCase())}
                      </p>
                      <p className="text-xs text-gray-400 mt-0.5">
                        {log.entity_type} {log.entity_id?.slice(0, 8)}…
                      </p>
                    </div>
                    <time className="text-xs text-gray-400 whitespace-nowrap">
                      {format(new Date(log.created_at), 'dd MMM · HH:mm')}
                    </time>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
