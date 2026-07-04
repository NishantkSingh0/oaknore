import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { projectApi, routingApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import StatusBadge from '@/components/ui/StatusBadge'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { ArrowLeft, GitBranch, Layers, AlertCircle, MessageSquare, FileText, History, ExternalLink } from 'lucide-react'
import { format } from 'date-fns'

function InfoRow({ label, value }: { label: string; value?: string | number | null }) {
  if (!value && value !== 0) return null
  return (
    <div className="flex gap-2 py-1.5 border-b border-gray-50 last:border-0">
      <span className="w-40 text-xs text-gray-400 flex-shrink-0">{label}</span>
      <span className="text-sm text-gray-800 break-all">{value}</span>
    </div>
  )
}

export default function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const { canAccess } = useAuth()

  const { data: pRes, isLoading } = useQuery({
    queryKey: ['project', projectId],
    queryFn: () => projectApi.get(projectId!),
  })
  const { data: routingsRes } = useQuery({
    queryKey: ['routings', projectId],
    queryFn: () => routingApi.list(projectId!),
    enabled: !!projectId,
  })

  const p = pRes?.data?.data
  const routings = routingsRes?.data?.data ?? []
  const activeRouting = routings.find(r => r.status === 'ACTIVE')

  if (isLoading) return <LoadingSpinner fullScreen />
  if (!p) return <div className="p-8 text-gray-500">Project not found.</div>

  const quickLinks = [
    { to: `/projects/${projectId}/routing`, icon: GitBranch, label: 'Routing Builder', show: canAccess('SUPER_ADMIN','ADMIN','LAYER_2') },
    { to: `/projects/${projectId}/tasks`, icon: Layers, label: 'Task Board', show: true },
    { to: `/projects/${projectId}/issues`, icon: AlertCircle, label: 'Issues', show: true },
    { to: `/projects/${projectId}/queries`, icon: MessageSquare, label: 'Queries', show: true },
    { to: `/projects/${projectId}/reports`, icon: FileText, label: 'Reports', show: true },
    { to: `/projects/${projectId}/timeline`, icon: History, label: 'Timeline', show: true },
  ]

  return (
    <div className="max-w-5xl space-y-5">
      {/* Header */}
      <div className="flex items-start gap-3">
        <button onClick={() => navigate('/projects')} className="btn-ghost btn-sm p-2 mt-0.5"><ArrowLeft size={16} /></button>
        <div className="flex-1">
          <div className="flex items-center gap-3 flex-wrap">
            <h1 className="page-title">{p.name}</h1>
            <StatusBadge status={p.status} />
            <span className="badge badge-gray">v{p.current_revision}</span>
          </div>
          <p className="text-sm text-gray-500 mt-1">{p.po_number} · {p.client_name}</p>
        </div>
        {canAccess('SUPER_ADMIN','ADMIN') && (
          <button onClick={() => navigate(`/projects/${projectId}/routing`)} className="btn-primary btn-sm">
            <GitBranch size={14} /> {activeRouting ? 'Edit Routing' : 'Create Routing'}
          </button>
        )}
      </div>

      {/* Quick links */}
      <div className="grid grid-cols-3 md:grid-cols-6 gap-2">
        {quickLinks.filter(l => l.show).map(({ to, icon: Icon, label }) => (
          <Link key={to} to={to} className="card flex flex-col items-center gap-1.5 p-3 hover:border-brand-300 transition-colors text-center">
            <Icon size={18} className="text-brand-600" />
            <span className="text-xs font-medium text-gray-700">{label}</span>
          </Link>
        ))}
      </div>

      <div className="grid md:grid-cols-2 gap-5">
        {/* Project details */}
        <div className="card card-body space-y-0">
          <h2 className="section-title mb-3">Project Details</h2>
          <InfoRow label="Client" value={p.client_name} />
          <InfoRow label="Client Contact" value={p.client_contact} />
          <InfoRow label="Quantity" value={p.quantity} />
          <InfoRow label="Dimensions" value={p.dimensions} />
          <InfoRow label="Material" value={p.material_details} />
          <InfoRow label="Color" value={p.color_details} />
          <InfoRow label="Upholstery" value={p.upholstery} />
          <InfoRow label="Finish" value={p.finish} />
          <InfoRow label="Delivery Date" value={p.delivery_date ? format(new Date(p.delivery_date), 'dd MMM yyyy') : undefined} />
          <InfoRow label="Delivery Address" value={p.delivery_address} />
          <InfoRow label="Specifications" value={p.specifications} />
          <InfoRow label="Remarks" value={p.remarks} />
        </div>

        {/* File links + routing summary */}
        <div className="space-y-4">
          <div className="card card-body">
            <h2 className="section-title mb-3">File Links</h2>
            {[
              { label: 'Cover Image', url: p.cover_image_url },
              { label: 'CAD Files', url: p.cad_files_url },
              { label: 'Drawings', url: p.drawings_url },
              { label: 'Job Cards', url: p.job_cards_url },
              { label: 'Render Files', url: p.render_files_url },
            ].filter(f => f.url).map(f => (
              <a key={f.label} href={f.url!} target="_blank" rel="noopener noreferrer"
                className="flex items-center gap-2 py-1.5 text-sm text-brand-600 hover:text-brand-800">
                <ExternalLink size={13} /> {f.label}
              </a>
            ))}
            {!p.cover_image_url && !p.cad_files_url && !p.drawings_url && (
              <p className="text-sm text-gray-400">No file links attached.</p>
            )}
          </div>

          <div className="card card-body">
            <h2 className="section-title mb-3">Routing Summary</h2>
            {routings.length === 0 ? (
              <p className="text-sm text-gray-400">No routing created yet.</p>
            ) : (
              <div className="space-y-2">
                {routings.map(r => (
                  <div key={r.id} className="flex items-center justify-between text-sm">
                    <span className="text-gray-700">Version {r.version}</span>
                    <StatusBadge status={r.status} />
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
