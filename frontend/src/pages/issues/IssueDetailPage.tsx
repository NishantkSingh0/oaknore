import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { issueApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import StatusBadge from '@/components/ui/StatusBadge'
import toast from 'react-hot-toast'
import { ArrowLeft } from 'lucide-react'
import { format } from 'date-fns'
import { useState } from 'react'

export default function IssueDetailPage() {
  const { issueId } = useParams<{ issueId: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { canAccess } = useAuth()
  const [resolutionNote, setResolutionNote] = useState('')

  const { data, isLoading } = useQuery({ queryKey: ['issue', issueId], queryFn: () => issueApi.get(issueId!) })
  const issue = data?.data?.data

  const { mutate: review } = useMutation({
    mutationFn: ({ decision, notes }: { decision: string; notes?: string }) =>
      issueApi.review(issueId!, decision, notes),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['issue', issueId] }); toast.success('Reviewed') },
  })

  const { mutate: resolve } = useMutation({
    mutationFn: () => issueApi.resolve(issueId!, resolutionNote),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['issue', issueId] }); toast.success('Issue resolved') },
  })

  if (isLoading) return <LoadingSpinner fullScreen />
  if (!issue) return <div className="p-8 text-gray-500">Issue not found.</div>

  return (
    <div className="max-w-2xl space-y-5">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate(-1)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
        <div className="flex-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="page-title">{issue.title}</h1>
            <StatusBadge status={issue.status} />
          </div>
          <p className="text-xs text-gray-400 mt-0.5">{issue.issue_type.replace(/_/g,' ')} · {format(new Date(issue.created_at),'dd MMM yyyy')}</p>
        </div>
      </div>

      <div className="card card-body space-y-3">
        <p className="text-sm text-gray-700 whitespace-pre-wrap">{issue.description}</p>
        {issue.review_notes && (
          <div className="rounded-lg bg-blue-50 border border-blue-100 p-3">
            <p className="text-xs font-medium text-blue-700 mb-1">Review Notes</p>
            <p className="text-sm text-blue-800">{issue.review_notes}</p>
          </div>
        )}
        {issue.resolution_note && (
          <div className="rounded-lg bg-green-50 border border-green-100 p-3">
            <p className="text-xs font-medium text-green-700 mb-1">Resolution</p>
            <p className="text-sm text-green-800">{issue.resolution_note}</p>
          </div>
        )}
      </div>

      {/* Layer 2 actions */}
      {canAccess('SUPER_ADMIN','ADMIN','LAYER_2') && issue.status === 'OPEN' && (
        <div className="card card-body space-y-3">
          <h2 className="section-title">Review Issue</h2>
          <div className="flex gap-3">
            <button onClick={() => review({ decision: 'approve' })} className="btn-primary">Approve</button>
            <button onClick={() => review({ decision: 'reject' })} className="btn-danger">Reject</button>
          </div>
        </div>
      )}

      {/* Layer 3 resolve */}
      {canAccess('LAYER_3') && issue.status === 'APPROVED' && (
        <div className="card card-body space-y-3">
          <h2 className="section-title">Resolve Issue</h2>
          <textarea value={resolutionNote} onChange={e => setResolutionNote(e.target.value)}
            rows={2} className="input resize-none" placeholder="Describe how the issue was resolved…" />
          <button onClick={() => resolve()} className="btn-primary">Mark Resolved</button>
        </div>
      )}
    </div>
  )
}
