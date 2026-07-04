import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { taskApi, subtaskApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import StatusBadge from '@/components/ui/StatusBadge'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, CheckCircle2, Circle, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'

export default function TaskDetailPage() {
  const { taskId } = useParams<{ taskId: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [addSubOpen, setAddSubOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['task', taskId],
    queryFn: () => taskApi.get(taskId!),
    refetchInterval: 15_000,
  })
  const task = data?.data?.data
  const subtasks = task?.subtasks ?? []
  const completed = subtasks.filter(s => s.status === 'COMPLETED').length

  const { mutate: complete } = useMutation({
    mutationFn: (id: string) => subtaskApi.complete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task', taskId] }); toast.success('Subtask completed') },
    onError: () => toast.error('Failed'),
  })
  const { mutate: deleteSubtask } = useMutation({
    mutationFn: (id: string) => subtaskApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', taskId] }),
  })
  const { mutate: updateTaskStatus } = useMutation({
    mutationFn: (status: string) => taskApi.updateStatus(taskId!, status),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task', taskId] }); toast.success('Status updated') },
  })

  const { register, handleSubmit, reset } = useForm<{ title: string; description: string; is_required: boolean }>()
  const { mutate: addSub, isPending: addingSubtask } = useMutation({
    mutationFn: (d: object) => subtaskApi.create(taskId!, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['task', taskId] }); setAddSubOpen(false); reset() },
    onError: () => toast.error('Failed to add subtask'),
  })

  if (isLoading) return <LoadingSpinner fullScreen />
  if (!task) return <div className="p-8 text-gray-500">Task not found.</div>

  return (
    <div className="max-w-2xl space-y-5">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate(-1)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="page-title">Department Task</h1>
            <StatusBadge status={task.status} />
          </div>
          <p className="text-sm text-gray-400 mt-0.5">Priority: {task.priority >= 3 ? 'High' : task.priority === 2 ? 'Medium' : 'Low'}</p>
        </div>
        <select value={task.status} onChange={e => updateTaskStatus(e.target.value)} className="input w-40 text-sm">
          {['PENDING','IN_PROGRESS','HOLD','ISSUE_HOLD','COMPLETED'].map(s => (
            <option key={s} value={s}>{s.replace(/_/g,' ')}</option>
          ))}
        </select>
      </div>

      {/* Dates */}
      {(task.start_date || task.due_date) && (
        <div className="card card-body flex gap-6">
          {task.start_date && <div><p className="text-xs text-gray-400">Start</p><p className="text-sm font-medium">{task.start_date}</p></div>}
          {task.due_date && <div><p className="text-xs text-gray-400">Due</p><p className="text-sm font-medium">{task.due_date}</p></div>}
          {task.dates_frozen && <span className="badge badge-gray self-end">Dates frozen</span>}
        </div>
      )}

      {/* Subtasks */}
      <div className="card">
        <div className="card-header flex items-center justify-between">
          <div>
            <h2 className="section-title">Subtasks</h2>
            <p className="text-xs text-gray-400 mt-0.5">{completed}/{subtasks.length} completed</p>
          </div>
          <button onClick={() => setAddSubOpen(true)} className="btn-primary btn-sm"><Plus size={14} /> Add</button>
        </div>
        {/* Progress bar */}
        {subtasks.length > 0 && (
          <div className="px-6 py-3 border-b border-gray-100">
            <div className="h-1.5 w-full rounded-full bg-gray-200">
              <div className="h-full rounded-full bg-brand-500 transition-all" style={{ width: `${(completed/subtasks.length)*100}%` }} />
            </div>
          </div>
        )}
        <div className="divide-y divide-gray-50">
          {subtasks.length === 0 && <p className="px-6 py-8 text-center text-sm text-gray-400">No subtasks yet.</p>}
          {subtasks.map(sub => (
            <div key={sub.id} className="flex items-start gap-3 px-6 py-3">
              <button
                onClick={() => sub.status !== 'COMPLETED' && complete(sub.id)}
                disabled={sub.status === 'COMPLETED'}
                className="mt-0.5 shrink-0"
              >
                {sub.status === 'COMPLETED'
                  ? <CheckCircle2 size={18} className="text-green-500" />
                  : <Circle size={18} className="text-gray-300 hover:text-brand-400 transition-colors" />}
              </button>
              <div className="flex-1 min-w-0">
                <p className={`text-sm font-medium ${sub.status === 'COMPLETED' ? 'line-through text-gray-400' : 'text-gray-800'}`}>{sub.title}</p>
                {sub.description && <p className="text-xs text-gray-400 mt-0.5">{sub.description}</p>}
                <div className="flex items-center gap-2 mt-1">
                  {sub.is_required && <span className="badge badge-red">Required</span>}
                  <StatusBadge status={sub.status} />
                </div>
              </div>
              <button onClick={() => deleteSubtask(sub.id)} className="btn-ghost btn-sm p-1 text-gray-300 hover:text-red-500"><Trash2 size={14} /></button>
            </div>
          ))}
        </div>
      </div>

      {task.notes && (
        <div className="card card-body">
          <h2 className="section-title mb-2">Notes</h2>
          <p className="text-sm text-gray-700 whitespace-pre-wrap">{task.notes}</p>
        </div>
      )}

      {/* Add subtask modal */}
      <Modal open={addSubOpen} onClose={() => setAddSubOpen(false)} title="Add Subtask" size="sm">
        <form onSubmit={handleSubmit(d => addSub(d))} className="space-y-4">
          <div>
            <label className="label">Title *</label>
            <input {...register('title', { required: true })} className="input" placeholder="Subtask title" />
          </div>
          <div>
            <label className="label">Description</label>
            <textarea {...register('description')} rows={2} className="input resize-none" />
          </div>
          <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
            <input type="checkbox" {...register('is_required')} className="rounded" defaultChecked />
            Required to complete task
          </label>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setAddSubOpen(false)} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={addingSubtask} className="btn-primary">{addingSubtask ? 'Adding…' : 'Add Subtask'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
