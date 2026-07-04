import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { projectApi, empApi, issueApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import StatusBadge from '@/components/ui/StatusBadge'
import { Search, FolderKanban, Users, AlertCircle, X } from 'lucide-react'
import { format } from 'date-fns'

type Tab = 'projects' | 'employees' | 'issues'

export default function SearchPage() {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [tab, setTab] = useState<Tab>('projects')
  const [debouncedQ, setDebouncedQ] = useState('')

  // Simple debounce
  const handleChange = useCallback((val: string) => {
    setQuery(val)
    const t = setTimeout(() => setDebouncedQ(val), 350)
    return () => clearTimeout(t)
  }, [])

  const enabled = debouncedQ.trim().length >= 2

  const { data: projRes, isFetching: searchingProjects } = useQuery({
    queryKey: ['search-projects', debouncedQ],
    queryFn: () => projectApi.list({ q: debouncedQ, limit: 20 }),
    enabled: enabled && tab === 'projects',
  })

  const { data: empRes, isFetching: searchingEmps } = useQuery({
    queryKey: ['search-employees', debouncedQ],
    queryFn: () => empApi.list({ limit: 20 }),
    enabled: enabled && tab === 'employees',
  })

  const projects = projRes?.data?.data ?? []
  const employees = (empRes?.data?.data ?? []).filter(e =>
    `${e.first_name} ${e.last_name} ${e.email}`.toLowerCase().includes(debouncedQ.toLowerCase())
  )

  const isSearching = searchingProjects || searchingEmps
  const showResults = enabled

  const TABS: { key: Tab; label: string; icon: React.ElementType }[] = [
    { key: 'projects', label: 'Projects', icon: FolderKanban },
    { key: 'employees', label: 'Employees', icon: Users },
  ]

  return (
    <div className="max-w-3xl space-y-5">
      <h1 className="page-title">Search</h1>

      {/* Search input */}
      <div className="relative">
        <Search size={18} className="absolute left-4 top-1/2 -translate-y-1/2 text-gray-400" />
        <input
          autoFocus
          value={query}
          onChange={e => handleChange(e.target.value)}
          placeholder="Search projects, PO numbers, clients, employees…"
          className="input pl-11 pr-10 py-3 text-base shadow-sm"
        />
        {query && (
          <button onClick={() => { setQuery(''); setDebouncedQ('') }}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600">
            <X size={16} />
          </button>
        )}
      </div>

      {!enabled && (
        <div className="text-center py-16 text-gray-400">
          <Search size={40} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">Type at least 2 characters to search</p>
        </div>
      )}

      {enabled && (
        <>
          {/* Tabs */}
          <div className="flex gap-1 border-b border-gray-200">
            {TABS.map(({ key, label, icon: Icon }) => (
              <button key={key} onClick={() => setTab(key)}
                className={`flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors -mb-px ${
                  tab === key
                    ? 'border-brand-600 text-brand-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}>
                <Icon size={15} /> {label}
              </button>
            ))}
          </div>

          {isSearching && <LoadingSpinner />}

          {/* Projects results */}
          {tab === 'projects' && !searchingProjects && (
            <>
              {projects.length === 0 ? (
                <p className="text-sm text-gray-400 py-8 text-center">No projects match "{debouncedQ}"</p>
              ) : (
                <div className="space-y-2">
                  <p className="text-xs text-gray-400">{projects.length} result{projects.length !== 1 ? 's' : ''}</p>
                  {projects.map(p => (
                    <div key={p.id}
                      onClick={() => navigate(`/projects/${p.id}`)}
                      className="card p-4 cursor-pointer hover:border-brand-300 transition-colors">
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap mb-1">
                            <p className="text-sm font-semibold text-gray-900">{p.name}</p>
                            <StatusBadge status={p.status} />
                          </div>
                          <div className="flex items-center gap-3 text-xs text-gray-500">
                            <span className="font-mono">{p.po_number}</span>
                            <span>·</span>
                            <span>{p.client_name}</span>
                            <span>·</span>
                            <span>Qty {p.quantity}</span>
                          </div>
                        </div>
                        <div className="text-right flex-shrink-0">
                          <p className="text-xs text-gray-400">Rev v{p.current_revision}</p>
                          {p.delivery_date && (
                            <p className="text-xs text-gray-400 mt-0.5">
                              {format(new Date(p.delivery_date), 'dd MMM yy')}
                            </p>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}

          {/* Employees results */}
          {tab === 'employees' && !searchingEmps && (
            <>
              {employees.length === 0 ? (
                <p className="text-sm text-gray-400 py-8 text-center">No employees match "{debouncedQ}"</p>
              ) : (
                <div className="space-y-2">
                  <p className="text-xs text-gray-400">{employees.length} result{employees.length !== 1 ? 's' : ''}</p>
                  {employees.map(e => (
                    <div key={e.id} className="card p-4 flex items-center gap-3">
                      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-100 text-sm font-semibold text-brand-700 uppercase">
                        {e.first_name[0]}{e.last_name[0]}
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-gray-900">{e.first_name} {e.last_name}</p>
                        <p className="text-xs text-gray-400">{e.email}</p>
                      </div>
                      <div className="flex items-center gap-2 flex-shrink-0">
                        <span className="badge badge-blue">{e.role.replace('_', ' ')}</span>
                        <span className={`badge ${e.is_active ? 'badge-green' : 'badge-gray'}`}>
                          {e.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </>
      )}
    </div>
  )
}
