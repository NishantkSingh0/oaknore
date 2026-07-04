import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { projectApi, routingApi, deptApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import StatusBadge from '@/components/ui/StatusBadge'
import toast from 'react-hot-toast'
import { ArrowLeft, Plus, Trash2, GitBranch, ChevronDown } from 'lucide-react'
import type { DependencyPolicy } from '@/types'

interface StepDraft {
  step_order: number
  label: string
  dependency_policy: DependencyPolicy
  department_ids: string[]
}

export default function RoutingBuilderPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const { data: pRes } = useQuery({ queryKey: ['project', projectId], queryFn: () => projectApi.get(projectId!) })
  const { data: deptRes } = useQuery({ queryKey: ['departments'], queryFn: () => deptApi.list() })
  const { data: routingsRes, isLoading } = useQuery({ queryKey: ['routings', projectId], queryFn: () => routingApi.list(projectId!) })

  const project = pRes?.data?.data
  const depts = (deptRes?.data?.data ?? []).filter(d => d.layer === 'LAYER_3' && d.is_active)
  const routings = routingsRes?.data?.data ?? []
  const activeRouting = routings.find(r => r.status === 'ACTIVE')

  const [steps, setSteps] = useState<StepDraft[]>([
    { step_order: 1, label: '', dependency_policy: 'REQUIRE_ALL', department_ids: [] },
  ])
  const [notes, setNotes] = useState('')

  const addStep = () => setSteps(s => [...s, { step_order: s.length + 1, label: '', dependency_policy: 'REQUIRE_ALL', department_ids: [] }])
  const removeStep = (i: number) => setSteps(s => s.filter((_, idx) => idx !== i).map((st, idx) => ({ ...st, step_order: idx + 1 })))
  const toggleDept = (stepIdx: number, deptId: string) =>
    setSteps(s => s.map((st, i) => i !== stepIdx ? st : {
      ...st,
      department_ids: st.department_ids.includes(deptId)
        ? st.department_ids.filter(id => id !== deptId)
        : [...st.department_ids, deptId],
    }))

  const { mutate, isPending } = useMutation({
    mutationFn: () => routingApi.create(projectId!, { notes, steps }),
    onSuccess: () => {
      toast.success('Routing created')
      qc.invalidateQueries({ queryKey: ['routings', projectId] })
      navigate(`/projects/${projectId}/tasks`)
    },
    onError: () => toast.error('Failed to create routing'),
  })

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-3xl space-y-5">
      <div className="flex items-center gap-3">
        <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-ghost btn-sm p-2"><ArrowLeft size={16} /></button>
        <div>
          <h1 className="page-title">Routing Builder</h1>
          {project && <p className="text-sm text-gray-500">{project.name}</p>}
        </div>
      </div>

      {/* Existing routings */}
      {routings.length > 0 && (
        <div className="card card-body space-y-2">
          <h2 className="section-title">Routing History</h2>
          {routings.map(r => (
            <div key={r.id} className="flex items-center justify-between py-1.5 text-sm border-b border-gray-50 last:border-0">
              <span className="text-gray-700 font-medium">Version {r.version}</span>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-400">{r.steps?.length ?? 0} steps</span>
                <StatusBadge status={r.status} />
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Builder */}
      <div className="card card-body space-y-5">
        <div className="flex items-center justify-between">
          <h2 className="section-title">
            {activeRouting ? `New Version (v${activeRouting.version + 1})` : 'Create Routing (v1)'}
          </h2>
          <span className="badge badge-blue flex items-center gap-1"><GitBranch size={11} /> Sequential + Parallel</span>
        </div>

        <div>
          <label className="label">Routing Notes</label>
          <input value={notes} onChange={e => setNotes(e.target.value)} className="input" placeholder="Optional notes…" />
        </div>

        {/* Steps */}
        <div className="space-y-4">
          {steps.map((step, i) => (
            <div key={i} className="rounded-xl border border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-brand-600 uppercase tracking-wide">Step {step.step_order}</span>
                {steps.length > 1 && (
                  <button onClick={() => removeStep(i)} className="btn-ghost btn-sm p-1.5 text-red-500"><Trash2 size={14} /></button>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Label</label>
                  <input value={step.label} onChange={e => setSteps(s => s.map((st, idx) => idx === i ? { ...st, label: e.target.value } : st))}
                    className="input" placeholder="e.g. Metal + Carpentry" />
                </div>
                <div>
                  <label className="label">Gate Policy</label>
                  <div className="relative">
                    <select value={step.dependency_policy}
                      onChange={e => setSteps(s => s.map((st, idx) => idx === i ? { ...st, dependency_policy: e.target.value as DependencyPolicy } : st))}
                      className="input appearance-none pr-8">
                      <option value="REQUIRE_ALL">Require All</option>
                      <option value="REQUIRE_ANY">Require Any</option>
                    </select>
                    <ChevronDown size={14} className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  </div>
                </div>
              </div>
              <div>
                <label className="label">Departments (select one or more for parallel)</label>
                <div className="flex flex-wrap gap-2 mt-1">
                  {depts.map(d => (
                    <button key={d.id} type="button"
                      onClick={() => toggleDept(i, d.id)}
                      className={`rounded-lg px-3 py-1.5 text-xs font-medium border transition-colors ${
                        step.department_ids.includes(d.id)
                          ? 'bg-brand-600 text-white border-brand-600'
                          : 'bg-white text-gray-600 border-gray-200 hover:border-brand-300'
                      }`}>
                      {d.name}
                    </button>
                  ))}
                  {depts.length === 0 && <p className="text-xs text-gray-400">No Layer 3 departments found.</p>}
                </div>
              </div>
            </div>
          ))}
        </div>

        <button onClick={addStep} className="btn-secondary w-full">
          <Plus size={15} /> Add Step
        </button>

        <div className="flex justify-end gap-3 pt-2 border-t border-gray-100">
          <button onClick={() => navigate(`/projects/${projectId}`)} className="btn-secondary">Cancel</button>
          <button onClick={() => mutate()} disabled={isPending || steps.some(s => s.department_ids.length === 0)} className="btn-primary">
            {isPending ? 'Publishing…' : 'Publish Routing'}
          </button>
        </div>
      </div>
    </div>
  )
}
