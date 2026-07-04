import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, FolderKanban } from 'lucide-react'
import { projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import StatusBadge from '@/components/ui/StatusBadge'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import { format } from 'date-fns'
import ProjectCreatePage from './ProjectCreatePage'
import type { ProjectStatus } from '@/types'

const STATUSES: { value: ProjectStatus | ''; label: string }[] = [
  { value: '', label: 'All' },
  { value: 'CREATED', label: 'Created' },
  { value: 'ROUTING', label: 'Routing' },
  { value: 'IN_PROGRESS', label: 'In Progress' },
  { value: 'COMPLETED', label: 'Completed' },
  { value: 'ARCHIVED', label: 'Archived' },
]

export default function ProjectListPage() {
  const navigate = useNavigate()
  const [showCreate, setShowCreate] = useState(false)
  const { canAccess } = useAuth()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<ProjectStatus | ''>('')
  const [page, setPage] = useState(1)

  const { data, isLoading } = useQuery({
    queryKey: ['projects', search, status, page],
    queryFn: () => projectApi.list({ q: search || undefined, status: status || undefined, page, limit: 20 }),
  })

  const projects = data?.data?.data ?? []
  const meta = data?.data?.meta

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="page-title">Projects</h1>
        {canAccess('SUPER_ADMIN', 'ADMIN') && (
        <button onClick={() => setShowCreate(true)} className="btn-primary">
            <Plus size={16}/>
            New Project
        </button>
        )}
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-52">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1) }}
            placeholder="Search PO, client, project name…"
            className="input pl-9"
          />
        </div>
        <div className="flex gap-1.5">
          {STATUSES.map(s => (
            <button
              key={s.value}
              onClick={() => { setStatus(s.value); setPage(1) }}
              className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                status === s.value ? 'bg-brand-600 text-white' : 'bg-white border border-gray-200 text-gray-600 hover:border-brand-300'
              }`}
            >
              {s.label}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      {isLoading ? <LoadingSpinner /> : (
        <>
          {projects.length === 0 ? (
            <EmptyState icon={FolderKanban} title="No projects found" description="Create your first project to get started." action={canAccess('SUPER_ADMIN','ADMIN') ? <button onClick={() => navigate('/projects/new')} className="btn-primary btn-sm">New Project</button> : undefined} />
          ) : (
            <div className="table-container">
              <table className="table">
                <thead>
                  <tr>
                    <th>Project Name</th>
                    <th>PO Number</th>
                    <th>Client</th>
                    <th>Qty</th>
                    <th>Status</th>
                    <th>Delivery</th>
                    <th>Rev</th>
                    <th>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {projects.map(p => (
                    <tr key={p.id} className="cursor-pointer" onClick={() => navigate(`/projects/${p.id}`)}>
                      <td className="font-medium text-gray-900 max-w-48 truncate">{p.name}</td>
                      <td className="text-gray-500 font-mono text-xs">{p.po_number}</td>
                      <td>{p.client_name}</td>
                      <td className="text-center">{p.quantity}</td>
                      <td><StatusBadge status={p.status} /></td>
                      <td className="text-xs text-gray-500">{p.delivery_date ? format(new Date(p.delivery_date), 'dd MMM yy') : '—'}</td>
                      <td className="text-center text-xs text-gray-400">v{p.current_revision}</td>
                      <td className="text-xs text-gray-400">{format(new Date(p.updated_at), 'dd MMM yy')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {meta && meta.total_pages > 1 && (
            <div className="flex items-center justify-between text-sm text-gray-500">
              <span>{meta.total} projects</span>
              <div className="flex gap-2">
                <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} className="btn-secondary btn-sm disabled:opacity-40">Prev</button>
                <span className="px-2 py-1">Page {page} / {meta.total_pages}</span>
                <button disabled={page >= meta.total_pages} onClick={() => setPage(p => p + 1)} className="btn-secondary btn-sm disabled:opacity-40">Next</button>
              </div>
            </div>
          )}
        </>
      )}
      {showCreate && (
        <ProjectCreatePage
            onClose={() => setShowCreate(false)}
        />
      )}
    </div>
  )
}
