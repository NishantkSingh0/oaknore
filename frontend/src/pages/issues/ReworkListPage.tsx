import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { reworkApi, deptApi, projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import StatusBadge from '@/components/ui/StatusBadge'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, RefreshCw } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { format } from 'date-fns'

export default function ReworkListPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { canAccess } = useAuth()
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [reviewOpen, setReviewOpen] = useState<string | null>(null)

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data, isLoading } = useQuery({ queryKey: ['reworks', projectId], queryFn: () => reworkApi.list(projectId!) })
  const { data: deptRes } = useQuery({ queryKey: ['departments'], queryFn: () => deptApi.list() })

  const project = pRes?.data?.data
  const reworks = data?.data?.data ?? []
  const depts = deptRes?.data?.data ?? []
  const getDeptName = (id: string) => depts.find(d => d.id === id)?.name ?? id.slice(0, 8)

  const { register: regCreate, handleSubmit: submitCreate, reset: resetCreate } = useForm<{
    originating_task_id: string; target_dept_id: string; reason: string
  }>()
  const { register: regReview, handleSubmit: submitReview, reset: resetReview } = useForm<{
    notes: string
  }>()

  const { mutate: create, isPending: creating } = useMutation({
    mutationFn: (d: object) => reworkApi.create(projectId!, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['reworks', projectId] }); setCreateOpen(false); resetCreate(); toast.success('Rework requested') },
    onError: () => toast.error('Failed'),
  })

  const { mutate: review, isPending: reviewing } = useMutation({
    mutationFn: ({ id, decision, notes }: { id: string; decision: string; notes: string }) =>
      reworkApi.review(id, { decision, notes }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['reworks', projectId] }); setReviewOpen(null); resetReview(); toast.success('Rework reviewed') },
    onError: () => toast.error('Failed'),
  })

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-3xl space-y-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
          <div>
            <h1 className="page-title">Rework Requests</h1>
            {project && <p className="text-sm text-gray-500">{project.name}</p>}
          </div>
        </div>
        {canAccess('LAYER_3') && (
          <button onClick={() => setCreateOpen(true)} className="btn-primary btn-sm"><Plus size={14} /> Request Rework</button>
        )}
      </div>

      {reworks.length === 0 ? (
        <EmptyState icon={RefreshCw} title="No rework requests" description="Rework requests will appear here when raised." />
      ) : (
        <div className="space-y-3">
          {reworks.map(rw => (
            <div key={rw.id} className="card p-4 space-y-2">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <StatusBadge status={rw.status} />
                    <span className="text-xs text-gray-400">{format(new Date(rw.created_at),'dd MMM yyyy')}</span>
                  </div>
                  <p className="text-sm font-medium text-gray-800">Send back to: <span className="text-brand-600">{getDeptName(rw.target_dept_id)}</span></p>
                  <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{rw.reason}</p>
                </div>
              </div>
              {rw.review_notes && (
                <p className="text-xs text-gray-500 bg-gray-50 rounded-lg px-3 py-2">{rw.review_notes}</p>
              )}
              {canAccess('SUPER_ADMIN','ADMIN','LAYER_2') && rw.status === 'PENDING' && (
                <div className="flex gap-2 pt-2 border-t border-gray-100">
                  <button onClick={() => setReviewOpen(rw.id)} className="btn-primary btn-sm">Review</button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Create rework modal */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="Request Rework" size="md">
        <form onSubmit={submitCreate(d => create(d))} className="space-y-4">
          <div>
            <label className="label">Target Department (to send work back to) *</label>
            <select {...regCreate('target_dept_id', { required: true })} className="input">
              <option value="">Select department…</option>
              {depts.filter(d => d.layer === 'LAYER_3').map(d => <option key={d.id} value={d.id}>{d.name}</option>)}
            </select>
          </div>
          <div>
            <label className="label">Originating Task ID *</label>
            <input {...regCreate('originating_task_id', { required: true })} className="input" placeholder="Paste task UUID" />
          </div>
          <div>
            <label className="label">Reason *</label>
            <textarea {...regCreate('reason', { required: true })} rows={3} className="input resize-none" placeholder="Explain why rework is needed…" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setCreateOpen(false)} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={creating} className="btn-primary">{creating ? 'Requesting…' : 'Submit Request'}</button>
          </div>
        </form>
      </Modal>

      {/* Review modal */}
      <Modal open={!!reviewOpen} onClose={() => setReviewOpen(null)} title="Review Rework Request" size="sm">
        <form onSubmit={submitReview(d => review({ id: reviewOpen!, decision: 'approve', notes: d.notes }))} className="space-y-4">
          <div>
            <label className="label">Review Notes</label>
            <textarea {...regReview('notes')} rows={2} className="input resize-none" placeholder="Optional notes…" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => submitReview(d => review({ id: reviewOpen!, decision: 'reject', notes: d.notes }))()} className="btn-danger" disabled={reviewing}>Reject</button>
            <button type="submit" disabled={reviewing} className="btn-primary">{reviewing ? 'Processing…' : 'Approve'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
