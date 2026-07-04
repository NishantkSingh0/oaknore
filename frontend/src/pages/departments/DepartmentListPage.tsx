import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { deptApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { Building2, Plus, Pencil, ToggleLeft, ToggleRight } from 'lucide-react'
import { useForm } from 'react-hook-form'
import type { DepartmentLayer } from '@/types'

interface DeptForm { name: string; layer: DepartmentLayer; description: string }

export default function DepartmentListPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [layerFilter, setLayerFilter] = useState<DepartmentLayer | ''>('')

  const { data, isLoading } = useQuery({ queryKey: ['departments'], queryFn: () => deptApi.list() })
  const depts = (data?.data?.data ?? []).filter(d => !layerFilter || d.layer === layerFilter)

  const { register, handleSubmit, reset, setValue } = useForm<DeptForm>()

  const { mutate: create, isPending: creating } = useMutation({
    mutationFn: (d: DeptForm) => deptApi.create(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['departments'] }); setCreateOpen(false); reset(); toast.success('Department created') },
    onError: () => toast.error('Failed'),
  })

  const { mutate: update, isPending: updating } = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<DeptForm> }) => deptApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['departments'] }); setEditId(null); reset(); toast.success('Updated') },
    onError: () => toast.error('Failed'),
  })

  const toggleActive = (id: string, current: boolean) =>
    update({ id, data: { is_active: !current } as any })

  const openEdit = (d: { id: string; name: string; description?: string; layer: DepartmentLayer }) => {
    setEditId(d.id)
    setValue('name', d.name)
    setValue('description', d.description ?? '')
    setValue('layer', d.layer)
  }

  const layer2 = depts.filter(d => d.layer === 'LAYER_2')
  const layer3 = depts.filter(d => d.layer === 'LAYER_3')

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="max-w-4xl space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="page-title">Departments</h1>
        <button onClick={() => setCreateOpen(true)} className="btn-primary btn-sm"><Plus size={14} /> New Department</button>
      </div>

      {/* Layer filter */}
      <div className="flex gap-2">
        {(['', 'LAYER_2', 'LAYER_3'] as const).map(l => (
          <button key={l} onClick={() => setLayerFilter(l)}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${layerFilter === l ? 'bg-brand-600 text-white' : 'bg-white border border-gray-200 text-gray-600 hover:border-brand-300'}`}>
            {l === '' ? 'All' : l === 'LAYER_2' ? 'Production Mgmt (L2)' : 'Execution (L3)'}
          </button>
        ))}
      </div>

      {depts.length === 0 && <EmptyState icon={Building2} title="No departments" description="Create departments to assign employees and build routings." />}

      {/* Layer 2 */}
      {(!layerFilter || layerFilter === 'LAYER_2') && layer2.length > 0 && (
        <div className="card">
          <div className="card-header"><h2 className="section-title text-brand-700">Production Management — Layer 2</h2></div>
          <div className="divide-y divide-gray-50">
            {layer2.map(d => (
              <div key={d.id} className="flex items-center justify-between px-6 py-3">
                <div>
                  <p className={`text-sm font-medium ${d.is_active ? 'text-gray-800' : 'text-gray-400 line-through'}`}>{d.name}</p>
                  {d.description && <p className="text-xs text-gray-400">{d.description}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <span className={`badge ${d.is_active ? 'badge-green' : 'badge-gray'}`}>{d.is_active ? 'Active' : 'Inactive'}</span>
                  <button onClick={() => openEdit(d)} className="btn-ghost btn-sm p-1.5"><Pencil size={13} /></button>
                  <button onClick={() => toggleActive(d.id, d.is_active)} className={`btn-ghost btn-sm p-1.5 ${d.is_active ? 'text-green-600' : 'text-gray-400'}`}>
                    {d.is_active ? <ToggleRight size={18} /> : <ToggleLeft size={18} />}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Layer 3 */}
      {(!layerFilter || layerFilter === 'LAYER_3') && layer3.length > 0 && (
        <div className="card">
          <div className="card-header"><h2 className="section-title text-purple-700">Execution Departments — Layer 3</h2></div>
          <div className="divide-y divide-gray-50">
            {layer3.map(d => (
              <div key={d.id} className="flex items-center justify-between px-6 py-3">
                <div>
                  <p className={`text-sm font-medium ${d.is_active ? 'text-gray-800' : 'text-gray-400 line-through'}`}>{d.name}</p>
                  {d.description && <p className="text-xs text-gray-400">{d.description}</p>}
                </div>
                <div className="flex items-center gap-2">
                  <span className={`badge ${d.is_active ? 'badge-green' : 'badge-gray'}`}>{d.is_active ? 'Active' : 'Inactive'}</span>
                  <button onClick={() => openEdit(d)} className="btn-ghost btn-sm p-1.5"><Pencil size={13} /></button>
                  <button onClick={() => toggleActive(d.id, d.is_active)} className={`btn-ghost btn-sm p-1.5 ${d.is_active ? 'text-green-600' : 'text-gray-400'}`}>
                    {d.is_active ? <ToggleRight size={18} /> : <ToggleLeft size={18} />}
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Create Modal */}
      <Modal open={createOpen} onClose={() => { setCreateOpen(false); reset() }} title="New Department" size="sm">
        <form onSubmit={handleSubmit(d => create(d))} className="space-y-4">
          <div>
            <label className="label">Name *</label>
            <input {...register('name', { required: true })} className="input" placeholder="e.g. Metal, Carpentry, Planning" />
          </div>
          <div>
            <label className="label">Layer *</label>
            <select {...register('layer', { required: true })} className="input">
              <option value="LAYER_2">Layer 2 — Production Management</option>
              <option value="LAYER_3">Layer 3 — Execution</option>
            </select>
          </div>
          <div>
            <label className="label">Description</label>
            <input {...register('description')} className="input" placeholder="Optional" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setCreateOpen(false); reset() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={creating} className="btn-primary">{creating ? 'Creating…' : 'Create'}</button>
          </div>
        </form>
      </Modal>

      {/* Edit Modal */}
      <Modal open={!!editId} onClose={() => { setEditId(null); reset() }} title="Edit Department" size="sm">
        <form onSubmit={handleSubmit(d => update({ id: editId!, data: d }))} className="space-y-4">
          <div>
            <label className="label">Name</label>
            <input {...register('name')} className="input" />
          </div>
          <div>
            <label className="label">Description</label>
            <input {...register('description')} className="input" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setEditId(null); reset() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={updating} className="btn-primary">{updating ? 'Saving…' : 'Save'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
