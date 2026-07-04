import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { FolderKanban, CheckCircle2, AlertCircle, Clock, ArrowRight, TrendingUp } from 'lucide-react'
import { projectApi } from '@/api/endpoints'
import { useAuth } from '@/context/AuthContext'
import StatusBadge from '@/components/ui/StatusBadge'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { format } from 'date-fns'

function StatCard({ icon: Icon, label, value, color }: { icon: any; label: string; value: number | string; color: string }) {
  return (
    <div className="card p-5 flex items-center gap-4">
      <div className={`rounded-xl p-3 ${color}`}>
        <Icon size={20} className="text-white" />
      </div>
      <div>
        <p className="text-xs text-gray-500 font-medium">{label}</p>
        <p className="text-2xl font-bold text-gray-900">{value}</p>
      </div>
    </div>
  )
}

export default function DashboardPage() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const { data: projectsRes, isLoading } = useQuery({
    queryKey: ['projects', 'dashboard'],
    queryFn: () => projectApi.list({ limit: 100 }),
  })

  const projects = projectsRes?.data?.data ?? []
  const active = projects.filter(p => p.status === 'IN_PROGRESS').length
  const created = projects.filter(p => p.status === 'CREATED').length
  const routing = projects.filter(p => p.status === 'ROUTING').length
  const completed = projects.filter(p => p.status === 'COMPLETED').length
  const recent = [...projects].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()).slice(0, 8)

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="page-title">Welcome back, {user?.first_name}</h1>
        <p className="text-sm text-gray-500 mt-0.5">Here's what's happening across your production floor.</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon={FolderKanban} label="Total Projects" value={projects.length} color="bg-brand-600" />
        <StatCard icon={TrendingUp} label="In Progress" value={active} color="bg-purple-500" />
        <StatCard icon={Clock} label="Pending Routing" value={created + routing} color="bg-yellow-500" />
        <StatCard icon={CheckCircle2} label="Completed" value={completed} color="bg-green-500" />
      </div>

      {/* Recent projects */}
      <div className="card">
        <div className="card-header flex items-center justify-between">
          <h2 className="section-title">Recent Projects</h2>
          <button onClick={() => navigate('/projects')} className="flex items-center gap-1 text-sm text-brand-600 hover:text-brand-700 font-medium">
            View all <ArrowRight size={14} />
          </button>
        </div>
        <div className="table-container rounded-none border-0">
          <table className="table">
            <thead>
              <tr>
                <th>Project</th>
                <th>PO Number</th>
                <th>Client</th>
                <th>Status</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {recent.length === 0 && (
                <tr><td colSpan={5} className="text-center py-8 text-gray-400">No projects yet</td></tr>
              )}
              {recent.map(p => (
                <tr key={p.id} onClick={() => navigate(`/projects/${p.id}`)} className="cursor-pointer">
                  <td className="font-medium text-gray-900">{p.name}</td>
                  <td className="text-gray-500">{p.po_number}</td>
                  <td>{p.client_name}</td>
                  <td><StatusBadge status={p.status} /></td>
                  <td className="text-gray-400 text-xs">{format(new Date(p.updated_at), 'dd MMM yyyy')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
