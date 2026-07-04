import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { projectApi, taskApi, deptApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import StatusBadge from '@/components/ui/StatusBadge'
import toast from 'react-hot-toast'
import { ArrowLeft, Layers, ChevronDown } from 'lucide-react'
import { format } from 'date-fns'
import type { TaskStatus } from '@/types'

const STATUSES: TaskStatus[] = ['PENDING', 'IN_PROGRESS', 'HOLD', 'ISSUE_HOLD', 'COMPLETED']

export default function TaskBoardPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data: tasksRes, isLoading } = useQuery({ queryKey: ['tasks', projectId], queryFn: () => taskApi.list(projectId!) })
  const { data: deptRes } = useQuery({ queryKey: ['departments'], queryFn: () => deptApi.list() })

  const tasks = tasksRes?.data?.data ?? []
  const depts = deptRes?.data?.data ?? []
  const project = pRes?.data?.data
  const getDeptName = (id: string) => depts.find(d => d.id === id)?.name ?? id.slice(0, 8)

  const { mutate: updateStatus } = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) => taskApi.updateStatus(id, status),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['tasks', projectId] }); toast.success('Status updated') },
    onError: () => toast.error('Update failed'),
  })

  if (isLoading) return <LoadingSpinner fullScreen />

  const grouped = STATUSES.reduce<Record<TaskStatus, typeof tasks>>((acc, s) => {
    acc[s] = tasks.filter(t => t.status === s)
    return acc
  }, {} as any)

  const COL_STYLE: Record<TaskStatus, string> = {
    PENDING: 'border-t-gray-400',
    IN_PROGRESS: 'border-t-brand-500',
    HOLD: 'border-t-yellow-400',
    ISSUE_HOLD: 'border-t-red-500',
    COMPLETED: 'border-t-green-500',
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
        <div>
          <h1 className="page-title">Task Board</h1>
          {project && <p className="text-sm text-gray-500">{project.name}</p>}
        </div>
      </div>

      {tasks.length === 0 ? (
        <EmptyState icon={Layers} title="No tasks yet" description="Create a routing to generate department tasks." />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4 overflow-x-auto pb-2">
          {STATUSES.map(status => (
            <div key={status} className={`rounded-xl border-t-4 ${COL_STYLE[status]} bg-gray-50 p-3 min-w-48`}>
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-semibold text-gray-600 uppercase tracking-wide">
                  {status.replace(/_/g, ' ')}
                </span>
                <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs font-medium text-gray-600">
                  {grouped[status].length}
                </span>
              </div>
              <div className="space-y-2">
                {grouped[status].map(task => (
                  <div key={task.id}
                    onClick={() => navigate(`/tasks/${task.id}`)}
                    className="card p-3 cursor-pointer hover:border-brand-300 transition-colors space-y-2">
                    <p className="text-sm font-medium text-gray-800 leading-snug">{getDeptName(task.department_id)}</p>
                    {task.due_date && (
                      <p className="text-xs text-gray-400">{format(new Date(task.due_date), 'dd MMM yyyy')}</p>
                    )}
                    <div className="flex items-center justify-between">
                      <span className={`text-xs font-medium ${task.priority >= 3 ? 'text-red-500' : task.priority === 2 ? 'text-yellow-600' : 'text-gray-400'}`}>
                        {task.priority >= 3 ? 'High' : task.priority === 2 ? 'Med' : 'Low'}
                      </span>
                      <div className="relative">
                        <select
                          value={task.status}
                          onClick={e => e.stopPropagation()}
                          onChange={e => updateStatus({ id: task.id, status: e.target.value as TaskStatus })}
                          className="text-xs border border-gray-200 rounded px-2 py-0.5 pr-5 bg-white appearance-none cursor-pointer"
                        >
                          {STATUSES.map(s => <option key={s} value={s}>{s.replace(/_/g,' ')}</option>)}
                        </select>
                        <ChevronDown size={10} className="pointer-events-none absolute right-1.5 top-1/2 -translate-y-1/2 text-gray-400" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
