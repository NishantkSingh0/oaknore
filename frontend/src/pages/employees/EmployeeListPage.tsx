import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { empApi, deptApi } from '@/api/endpoints'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import Modal from '@/components/ui/Modal'
import toast from 'react-hot-toast'
import { Users, Plus, Pencil, Key, ArrowRightLeft, ToggleLeft, ToggleRight } from 'lucide-react'
import { useForm } from 'react-hook-form'
import type { UserRole } from '@/types'

const ROLES: UserRole[] = ['ADMIN', 'LAYER_2', 'LAYER_3']

export default function EmployeeListPage() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [editId, setEditId] = useState<string | null>(null)
  const [resetId, setResetId] = useState<string | null>(null)
  const [transferId, setTransferId] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [deptFilter, setDeptFilter] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['employees', page, deptFilter],
    queryFn: () => empApi.list({ page, limit: 20, department_id: deptFilter || undefined }),
  })
  const { data: deptRes } = useQuery({ queryKey: ['departments'], queryFn: () => deptApi.list() })

  const employees = data?.data?.data ?? []
  const meta = data?.data?.meta
  const depts = deptRes?.data?.data ?? []
  const getDeptName = (id?: string) => id ? (depts.find(d => d.id === id)?.name ?? '—') : '—'

  const { register: regCreate, handleSubmit: submitCreate, reset: resetCreate, formState: { errors: errCreate } } = useForm<{
    first_name: string; last_name: string; email: string; password: string
    phone: string; employee_id: string; role: UserRole; department_id: string
  }>()
  const { register: regEdit, handleSubmit: submitEdit, reset: resetEdit, setValue: setEditVal } = useForm<{
    first_name: string; last_name: string; phone: string; is_active: boolean
  }>()
  const { register: regReset, handleSubmit: submitReset, reset: resetResetForm } = useForm<{ new_password: string }>()
  const { register: regTransfer, handleSubmit: submitTransfer, reset: resetTransfer } = useForm<{ department_id: string }>()

  const invalidate = () => qc.invalidateQueries({ queryKey: ['employees'] })

  const { mutate: create, isPending: creating } = useMutation({
    mutationFn: (d: object) => empApi.create(d),
    onSuccess: () => { invalidate(); setCreateOpen(false); resetCreate(); toast.success('Employee created') },
    onError: () => toast.error('Email may already exist'),
  })
  const { mutate: update, isPending: updating } = useMutation({
    mutationFn: ({ id, data }: { id: string; data: object }) => empApi.update(id, data),
    onSuccess: () => { invalidate(); setEditId(null); resetEdit(); toast.success('Updated') },
    onError: () => toast.error('Update failed'),
  })
  const { mutate: resetPwd, isPending: resetting } = useMutation({
    mutationFn: ({ id, new_password }: { id: string; new_password: string }) => empApi.resetPassword(id, new_password),
    onSuccess: () => { setResetId(null); resetResetForm(); toast.success('Password reset') },
    onError: () => toast.error('Failed'),
  })
  const { mutate: transfer, isPending: transferring } = useMutation({
    mutationFn: ({ id, department_id }: { id: string; department_id: string }) => empApi.transfer(id, department_id),
    onSuccess: () => { invalidate(); setTransferId(null); resetTransfer(); toast.success('Transferred') },
    onError: () => toast.error('Failed'),
  })

  const openEdit = (e: any) => {
    setEditId(e.id)
    setEditVal('first_name', e.first_name)
    setEditVal('last_name', e.last_name)
    setEditVal('phone', e.phone ?? '')
    setEditVal('is_active', e.is_active)
  }

  if (isLoading) return <LoadingSpinner fullScreen />

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h1 className="page-title">Employees</h1>
        <button onClick={() => setCreateOpen(true)} className="btn-primary btn-sm"><Plus size={14} /> Add Employee</button>
      </div>

      {/* Dept filter */}
      <div className="flex items-center gap-3 flex-wrap">
        <select value={deptFilter} onChange={e => { setDeptFilter(e.target.value); setPage(1) }} className="input w-56 text-sm">
          <option value="">All Departments</option>
          {depts.map(d => <option key={d.id} value={d.id}>{d.name}</option>)}
        </select>
        {meta && <span className="text-sm text-gray-500">{meta.total} employees</span>}
      </div>

      {employees.length === 0 ? (
        <EmptyState icon={Users} title="No employees" description="Add employees to assign them to departments and tasks." />
      ) : (
        <div className="table-container">
          <table className="table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Role</th>
                <th>Department</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {employees.map(emp => (
                <tr key={emp.id}>
                  <td>
                    <div className="flex items-center gap-2">
                      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-brand-100 text-xs font-semibold text-brand-700 uppercase">
                        {emp.first_name[0]}{emp.last_name[0]}
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">{emp.first_name} {emp.last_name}</p>
                        {emp.employee_id && <p className="text-xs text-gray-400">{emp.employee_id}</p>}
                      </div>
                    </div>
                  </td>
                  <td className="text-gray-500">{emp.email}</td>
                  <td>
                    <span className="badge badge-blue">{emp.role.replace('_',' ')}</span>
                  </td>
                  <td className="text-gray-500">{getDeptName(emp.department_id)}</td>
                  <td>
                    <span className={`badge ${emp.is_active ? 'badge-green' : 'badge-gray'}`}>
                      {emp.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td>
                    <div className="flex items-center gap-1">
                      <button title="Edit" onClick={() => openEdit(emp)} className="btn-ghost btn-sm p-1.5"><Pencil size={13} /></button>
                      <button title="Reset Password" onClick={() => setResetId(emp.id)} className="btn-ghost btn-sm p-1.5"><Key size={13} /></button>
                      <button title="Transfer Dept" onClick={() => setTransferId(emp.id)} className="btn-ghost btn-sm p-1.5"><ArrowRightLeft size={13} /></button>
                      <button title={emp.is_active ? 'Disable' : 'Enable'} onClick={() => update({ id: emp.id, data: { is_active: !emp.is_active } })}
                        className={`btn-ghost btn-sm p-1.5 ${emp.is_active ? 'text-green-600' : 'text-gray-400'}`}>
                        {emp.is_active ? <ToggleRight size={16} /> : <ToggleLeft size={16} />}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {meta && meta.total_pages > 1 && (
        <div className="flex items-center justify-end gap-2 text-sm">
          <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} className="btn-secondary btn-sm disabled:opacity-40">Prev</button>
          <span className="text-gray-500">Page {page} / {meta.total_pages}</span>
          <button disabled={page >= meta.total_pages} onClick={() => setPage(p => p + 1)} className="btn-secondary btn-sm disabled:opacity-40">Next</button>
        </div>
      )}

      {/* Create Employee */}
      <Modal open={createOpen} onClose={() => { setCreateOpen(false); resetCreate() }} title="Add Employee" size="md">
        <form onSubmit={submitCreate(d => create(d))} className="grid grid-cols-2 gap-4">
          <div>
            <label className="label">First Name *</label>
            <input {...regCreate('first_name', { required: true })} className="input" />
          </div>
          <div>
            <label className="label">Last Name *</label>
            <input {...regCreate('last_name', { required: true })} className="input" />
          </div>
          <div className="col-span-2">
            <label className="label">Email *</label>
            <input {...regCreate('email', { required: true })} type="email" className="input" />
            {errCreate.email && <p className="field-error">Required</p>}
          </div>
          <div>
            <label className="label">Password *</label>
            <input {...regCreate('password', { required: true, minLength: 8 })} type="password" className="input" placeholder="Min 8 chars" />
            {errCreate.password && <p className="field-error">Min 8 characters</p>}
          </div>
          <div>
            <label className="label">Employee ID</label>
            <input {...regCreate('employee_id')} className="input" placeholder="EMP-001" />
          </div>
          <div>
            <label className="label">Phone</label>
            <input {...regCreate('phone')} className="input" />
          </div>
          <div>
            <label className="label">Role *</label>
            <select {...regCreate('role', { required: true })} className="input">
              {ROLES.map(r => <option key={r} value={r}>{r.replace('_',' ')}</option>)}
            </select>
          </div>
          <div className="col-span-2">
            <label className="label">Department</label>
            <select {...regCreate('department_id')} className="input">
              <option value="">None</option>
              {depts.map(d => <option key={d.id} value={d.id}>{d.name} ({d.layer})</option>)}
            </select>
          </div>
          <div className="col-span-2 flex justify-end gap-3">
            <button type="button" onClick={() => { setCreateOpen(false); resetCreate() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={creating} className="btn-primary">{creating ? 'Creating…' : 'Create Employee'}</button>
          </div>
        </form>
      </Modal>

      {/* Edit Employee */}
      <Modal open={!!editId} onClose={() => { setEditId(null); resetEdit() }} title="Edit Employee" size="sm">
        <form onSubmit={submitEdit(d => update({ id: editId!, data: d }))} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div><label className="label">First Name</label><input {...regEdit('first_name')} className="input" /></div>
            <div><label className="label">Last Name</label><input {...regEdit('last_name')} className="input" /></div>
          </div>
          <div><label className="label">Phone</label><input {...regEdit('phone')} className="input" /></div>
          <label className="flex items-center gap-2 text-sm text-gray-700 cursor-pointer">
            <input type="checkbox" {...regEdit('is_active')} className="rounded" />
            Active
          </label>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setEditId(null); resetEdit() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={updating} className="btn-primary">{updating ? 'Saving…' : 'Save'}</button>
          </div>
        </form>
      </Modal>

      {/* Reset Password */}
      <Modal open={!!resetId} onClose={() => { setResetId(null); resetResetForm() }} title="Reset Password" size="sm">
        <form onSubmit={submitReset(d => resetPwd({ id: resetId!, new_password: d.new_password }))} className="space-y-4">
          <div>
            <label className="label">New Password *</label>
            <input {...regReset('new_password', { required: true, minLength: 8 })} type="password" className="input" placeholder="Min 8 characters" />
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setResetId(null); resetResetForm() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={resetting} className="btn-primary">{resetting ? 'Resetting…' : 'Reset Password'}</button>
          </div>
        </form>
      </Modal>

      {/* Transfer Department */}
      <Modal open={!!transferId} onClose={() => { setTransferId(null); resetTransfer() }} title="Transfer Department" size="sm">
        <form onSubmit={submitTransfer(d => transfer({ id: transferId!, department_id: d.department_id }))} className="space-y-4">
          <div>
            <label className="label">New Department *</label>
            <select {...regTransfer('department_id', { required: true })} className="input">
              <option value="">Select department…</option>
              {depts.map(d => <option key={d.id} value={d.id}>{d.name} ({d.layer})</option>)}
            </select>
          </div>
          <div className="flex justify-end gap-3">
            <button type="button" onClick={() => { setTransferId(null); resetTransfer() }} className="btn-secondary">Cancel</button>
            <button type="submit" disabled={transferring} className="btn-primary">{transferring ? 'Transferring…' : 'Transfer'}</button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
