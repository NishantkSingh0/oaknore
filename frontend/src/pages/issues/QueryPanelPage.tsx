import { useState, useRef, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { queryApi, projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import StatusBadge from '@/components/ui/StatusBadge'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, MessageSquare, Send, CheckCheck } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { format } from 'date-fns'

export default function QueryPanelPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const qc = useQueryClient()
  const [selected, setSelected] = useState<string | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [msgText, setMsgText] = useState('')
  const msgEndRef = useRef<HTMLDivElement>(null)

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data, isLoading } = useQuery({ queryKey: ['queries', projectId], queryFn: () => queryApi.list(projectId!), refetchInterval: 10_000 })
  const { data: detailRes } = useQuery({
    queryKey: ['query', selected],
    queryFn: () => queryApi.get(selected!),
    enabled: !!selected,
    refetchInterval: 8_000,
  })

  const queries = data?.data?.data ?? []
  const detail = detailRes?.data?.data
  const messages = detail?.messages ?? []

  useEffect(() => { msgEndRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages.length])

  const { register, handleSubmit, reset } = useForm<{ receiver_id: string; subject: string; message: string }>()

  const { mutate: create, isPending: creating } = useMutation({
    mutationFn: (d: object) => queryApi.create(projectId!, d),
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ['queries', projectId] })
      setCreateOpen(false); reset()
      setSelected(res.data.data?.id ?? null)
      toast.success('Query opened')
    },
    onError: () => toast.error('Failed to create query'),
  })

  const { mutate: sendMsg } = useMutation({
    mutationFn: (body: string) => queryApi.postMessage(selected!, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['query', selected] }); setMsgText('') },
  })

  const { mutate: resolveQuery } = useMutation({
    mutationFn: () => queryApi.resolve(selected!),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['query', selected] }); qc.invalidateQueries({ queryKey: ['queries', projectId] }); toast.success('Marked as resolved') },
  })

  const project = pRes?.data?.data

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="flex h-[calc(100vh-10rem)] gap-4">
      {/* Query list */}
      <div className="w-72 flex-shrink-0 flex flex-col card">
        <div className="card-header flex items-center justify-between">
          <div className="flex items-center gap-2">
            <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-1"><ArrowLeft size={14} /></button>
            <h2 className="section-title text-sm">Queries</h2>
          </div>
          <button onClick={() => setCreateOpen(true)} className="btn-primary btn-sm p-1.5"><Plus size={14} /></button>
        </div>
        <div className="flex-1 overflow-y-auto divide-y divide-gray-50">
          {queries.length === 0 && (
            <EmptyState icon={MessageSquare} title="No queries" description="Start a query to communicate." />
          )}
          {queries.map(q => (
            <button key={q.id} onClick={() => setSelected(q.id)}
              className={`w-full text-left px-4 py-3 hover:bg-gray-50 transition-colors ${selected === q.id ? 'bg-brand-50 border-r-2 border-brand-600' : ''}`}>
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs font-medium text-gray-800 truncate flex-1">{q.subject}</p>
                <StatusBadge status={q.status} />
              </div>
              <p className="text-xs text-gray-400 mt-0.5">{format(new Date(q.created_at),'dd MMM')}</p>
            </button>
          ))}
        </div>
      </div>

      {/* Message thread */}
      <div className="flex-1 flex flex-col card overflow-hidden">
        {!selected ? (
          <div className="flex-1 flex items-center justify-center">
            <EmptyState icon={MessageSquare} title="Select a query" description="Choose a query from the list to view messages." />
          </div>
        ) : (
          <>
            <div className="card-header flex items-center justify-between">
              <div>
                <p className="text-sm font-semibold text-gray-900">{detail?.subject ?? '…'}</p>
                {detail && <StatusBadge status={detail.status} />}
              </div>
              {detail && detail.status !== 'CLOSED' && (
                <button onClick={() => resolveQuery()} className="btn-secondary btn-sm gap-1">
                  <CheckCheck size={13} /> Mark Resolved
                </button>
              )}
            </div>
            <div className="flex-1 overflow-y-auto p-4 space-y-3">
              {messages.map(msg => {
                const isMine = msg.sender_id === user?.id
                return (
                  <div key={msg.id} className={`flex ${isMine ? 'justify-end' : 'justify-start'}`}>
                    <div className={`max-w-xs rounded-2xl px-4 py-2.5 ${isMine ? 'bg-brand-600 text-white rounded-br-sm' : 'bg-gray-100 text-gray-800 rounded-bl-sm'}`}>
                      <p className="text-sm whitespace-pre-wrap">{msg.body}</p>
                      <p className={`text-[10px] mt-1 ${isMine ? 'text-brand-200' : 'text-gray-400'}`}>
                        {format(new Date(msg.created_at),'HH:mm')}
                      </p>
                    </div>
                  </div>
                )
              })}
              <div ref={msgEndRef} />
            </div>
            {detail?.status !== 'CLOSED' && (
              <div className="border-t border-gray-100 p-3 flex gap-2">
                <input value={msgText} onChange={e => setMsgText(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey && msgText.trim()) { e.preventDefault(); sendMsg(msgText.trim()) } }}
                  className="input flex-1 text-sm" placeholder="Type a message… (Enter to send)" />
                <button onClick={() => msgText.trim() && sendMsg(msgText.trim())} disabled={!msgText.trim()} className="btn-primary p-2.5">
                  <Send size={15} />
                </button>
              </div>
            )}
          </>
        )}
      </div>

      {/* Create query modal */}
      <Modal open={createOpen} onClose={() => setCreateOpen(false)} title="New Query" size="sm">
        <form onSubmit={handleSubmit(d => create(d))} className="space-y-4">
          <div>
            <label className="label">Receiver User ID *</label>
            <input {...register('receiver_id', { required: true })} className="input" placeholder="UUID of adjacent layer user" />
          </div>
          <div>
            <label className="label">Subject *</label>
            <input {...register('subject', { required: true })} className="input" placeholder="Brief subject" />
          </div>
          <div>
            <label className="label">First Message *</label>
            <textarea {...register('message', { required: true })} rows={3} className="input resize-none" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => setCreateOpen(false)} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={creating} className="btn-primary">{creating ? 'Opening…' : 'Open Query'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
