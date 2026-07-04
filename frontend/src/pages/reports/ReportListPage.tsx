import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { reportApi, projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, FileText } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { format } from 'date-fns'

export default function ReportListPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { canAccess } = useAuth()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data, isLoading } = useQuery({
    queryKey: ['reports', projectId, from, to],
    queryFn: () => reportApi.list(projectId!, { from: from || undefined, to: to || undefined }),
  })

  const project = pRes?.data?.data
  const reports = data?.data?.data ?? []

  const { register, handleSubmit, reset } = useForm<{ description: string; report_date: string }>()

  const { mutate: create, isPending } = useMutation({
    mutationFn: (d: object) => reportApi.create(projectId!, d),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['reports', projectId] })
      setOpen(false); reset()
      toast.success('Daily report submitted')
    },
    onError: () => toast.error('Failed to submit report'),
  })

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-3xl space-y-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2">
            <ArrowLeft size={16} />
          </button>
          <div>
            <h1 className="page-title">Daily Reports</h1>
            {project && <p className="text-sm text-gray-500">{project.name}</p>}
          </div>
        </div>
        {canAccess('LAYER_3') && (
          <button onClick={() => setOpen(true)} className="btn-primary btn-sm">
            <Plus size={14} /> Submit Report
          </button>
        )}
      </div>

      {/* Date range filter */}
      <div className="flex items-center gap-3">
        <div>
          <label className="label">From</label>
          <input type="date" value={from} onChange={e => setFrom(e.target.value)} className="input text-sm" />
        </div>
        <div>
          <label className="label">To</label>
          <input type="date" value={to} onChange={e => setTo(e.target.value)} className="input text-sm" />
        </div>
        {(from || to) && (
          <button onClick={() => { setFrom(''); setTo('') }} className="btn-ghost btn-sm mt-5">Clear</button>
        )}
      </div>

      {reports.length === 0 ? (
        <EmptyState icon={FileText} title="No reports yet" description="Daily reports submitted by departments will appear here." />
      ) : (
        <div className="space-y-3">
          {reports.map(rep => (
            <div key={rep.id} className="card p-4 space-y-2">
              <div className="flex items-center justify-between">
                <time className="text-xs font-semibold text-brand-600 bg-brand-50 px-2 py-0.5 rounded-md">
                  {format(new Date(rep.report_date), 'EEEE, dd MMMM yyyy')}
                </time>
                <span className="text-xs text-gray-400">
                  {format(new Date(rep.created_at), 'HH:mm')}
                </span>
              </div>
              <p className="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">{rep.description}</p>
              {rep.attachments && rep.attachments.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-1">
                  {rep.attachments.map(f => (
                    <a key={f.id} href={f.url} target="_blank" rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-xs text-brand-600 hover:text-brand-800 bg-brand-50 px-2 py-1 rounded">
                      <FileText size={11} /> {f.file_name}
                    </a>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Submit report modal */}
      <Modal open={open} onClose={() => { setOpen(false); reset() }} title="Submit Daily Report" size="md">
        <form onSubmit={handleSubmit(d => create(d))} className="space-y-4">
          <div>
            <label className="label">Report Date</label>
            <input {...register('report_date')} type="date"
              defaultValue={new Date().toISOString().split('T')[0]}
              className="input" />
          </div>
          <div>
            <label className="label">Description *</label>
            <textarea {...register('description', { required: true })}
              rows={5} className="input resize-none"
              placeholder="Describe today's progress, completed work, any observations…" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setOpen(false); reset() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={isPending} className="btn-primary">
              {isPending ? 'Submitting…' : 'Submit Report'}
            </button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
