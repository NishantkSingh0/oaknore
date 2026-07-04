import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { issueApi, projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import StatusBadge from '@/components/ui/StatusBadge'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, AlertCircle } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { format } from 'date-fns'
import type { IssueType } from '@/types'

const ISSUE_TYPES: IssueType[] = ['MATERIAL_MISSING','DESIGN_CHANGE','ROUTING_REQUIRED','FULL_SCALE_REQUIREMENT','QUALITY_ISSUE','REWORK_REQUIRED','CUSTOM']

export default function IssueListPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { canAccess } = useAuth()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [statusFilter, setStatusFilter] = useState('')

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data, isLoading } = useQuery({
    queryKey: ['issues', projectId, statusFilter],
    queryFn: () => issueApi.list(projectId!, statusFilter ? { status: statusFilter } : {}),
  })

  const project = pRes?.data?.data
  const issues = data?.data?.data ?? []

  const { register, handleSubmit, reset, formState: { errors } } = useForm<{
    issue_type: IssueType; custom_type: string; title: string; description: string
  }>()

  const { mutate: create, isPending } = useMutation({
    mutationFn: (d: object) => issueApi.create(projectId!, d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['issues', projectId] }); setOpen(false); reset(); toast.success('Issue raised') },
    onError: () => toast.error('Failed to raise issue'),
  })

  const { mutate: review } = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: string }) => issueApi.review(id, decision),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['issues', projectId] }); toast.success('Issue reviewed') },
    onError: () => toast.error('Review failed'),
  })

  const STATUSES = ['', 'OPEN', 'PENDING_APPROVAL', 'APPROVED', 'REJECTED', 'RESOLVED', 'CLOSED']

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-4xl space-y-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
          <div>
            <h1 className="page-title">Issues</h1>
            {project && <p className="text-sm text-gray-500">{project.name}</p>}
          </div>
        </div>
        {canAccess('LAYER_3') && (
          <button onClick={() => setOpen(true)} className="btn-primary btn-sm"><Plus size={14} /> Raise Issue</button>
        )}
      </div>

      {/* Status filter */}
      <div className="flex gap-1.5 flex-wrap">
        {STATUSES.map(s => (
          <button key={s} onClick={() => setStatusFilter(s)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${statusFilter === s ? 'bg-brand-600 text-white' : 'bg-white border border-gray-200 text-gray-600 hover:border-brand-300'}`}>
            {s || 'All'}
          </button>
        ))}
      </div>

      {issues.length === 0 ? (
        <EmptyState icon={AlertCircle} title="No issues" description="All clear — no issues have been raised." />
      ) : (
        <div className="space-y-3">
          {issues.map(issue => (
            <div key={issue.id} className="card p-4 cursor-pointer hover:border-brand-300 transition-colors"
              onClick={() => navigate(`/issues/${issue.id}`)}>
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <span className="badge badge-gray text-xs">{issue.issue_type.replace(/_/g,' ')}</span>
                    <StatusBadge status={issue.status} />
                  </div>
                  <p className="text-sm font-semibold text-gray-900">{issue.title}</p>
                  <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{issue.description}</p>
                </div>
                <time className="text-xs text-gray-400 whitespace-nowrap">{format(new Date(issue.created_at), 'dd MMM')}</time>
              </div>
              {/* Layer 2 review buttons */}
              {canAccess('SUPER_ADMIN','ADMIN','LAYER_2') && issue.status === 'OPEN' && (
                <div className="flex gap-2 mt-3 pt-3 border-t border-gray-100" onClick={e => e.stopPropagation()}>
                  <button onClick={() => review({ id: issue.id, decision: 'approve' })} className="btn-primary btn-sm">Approve</button>
                  <button onClick={() => review({ id: issue.id, decision: 'reject' })} className="btn-danger btn-sm">Reject</button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Raise issue modal */}
      <Modal open={open} onClose={() => setOpen(false)} title="Raise Issue" size="md">
        <form onSubmit={handleSubmit(d => create(d))} className="space-y-4">
          <div>
            <label className="label">Issue Type *</label>
            <select {...register('issue_type', { required: true })} className="input">
              {ISSUE_TYPES.map(t => <option key={t} value={t}>{t.replace(/_/g,' ')}</option>)}
            </select>
          </div>
          <div>
            <label className="label">Title *</label>
            <input {...register('title', { required: 'Required' })} className="input" placeholder="Brief issue title" />
            {errors.title && <p className="field-error">{errors.title.message}</p>}
          </div>
          <div>
            <label className="label">Description *</label>
            <textarea {...register('description', { required: 'Required' })} rows={3} className="input resize-none" placeholder="Describe the issue…" />
            {errors.description && <p className="field-error">{errors.description.message}</p>}
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setOpen(false)} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={isPending} className="btn-primary">{isPending ? 'Raising…' : 'Raise Issue'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
