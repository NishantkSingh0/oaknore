import { useForm } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { projectApi } from '@/api/endpoints'
import toast from 'react-hot-toast'

interface FormValues {
  po_number: string; client_name: string; client_contact: string; name: string
  quantity: number; dimensions: string; specifications: string; material_details: string
  color_details: string; upholstery: string; finish: string; delivery_date: string
  delivery_address: string; remarks: string; cover_image_url: string
  cad_files_url: string; drawings_url: string; job_cards_url: string; render_files_url: string
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return <div><label className="label">{label}</label>{children}</div>
}

interface Props {
    onClose: () => void
}

export default function ProjectCreatePage({ onClose }: Props) {
  const { register, handleSubmit, formState: { errors } } = useForm<FormValues>({ defaultValues: { quantity: 1 } })

  const queryClient = useQueryClient()
  const { mutate, isPending } = useMutation({
    mutationFn: (data: FormValues) => projectApi.create(data),
    onSuccess: () => {
        toast.success('Project created')
        queryClient.invalidateQueries({ queryKey: ['projects']})
        onClose()
    },
    onError: () => toast.error('Failed to create project'),
  })

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-2xl shadow-2xl w-[95vw] max-w-6xl h-[90vh] overflow-hidden flex flex-col">
        <div className="flex items-center justify-between border-b px-6 py-4">
          <h1 className="text-xl font-semibold">
              Create Project
          </h1>
          <button onClick={onClose} className="btn-secondary">✕</button>
        </div>
        <form onSubmit={handleSubmit((d) => mutate(d))} className="flex-1 overflow-y-auto p-6 space-y-6">          
          {/* Core info */}
          <div className="card card-body grid grid-cols-1 md:grid-cols-2 gap-4">
            <h2 className="section-title col-span-full">Project Information</h2>
            <Field label="PO Number *">
              <input {...register('po_number', { required: 'Required' })} className="input" placeholder="PO-2024-001" />
              {errors.po_number && <p className="field-error">{errors.po_number.message}</p>}
            </Field>
            <Field label="Project Name *">
              <input {...register('name', { required: 'Required' })} className="input" placeholder="Custom Sofa Set" />
              {errors.name && <p className="field-error">{errors.name.message}</p>}
            </Field>
            <Field label="Client Name *">
              <input {...register('client_name', { required: 'Required' })} className="input" placeholder="Acme Corp" />
              {errors.client_name && <p className="field-error">{errors.client_name.message}</p>}
            </Field>
            <Field label="Client Contact">
              <input {...register('client_contact')} className="input" placeholder="+91 9876543210" />
            </Field>
            <Field label="Quantity *">
              <input {...register('quantity', { required: true, min: 1, valueAsNumber: true })} type="number" min={1} className="input" />
            </Field>
            <Field label="Delivery Date">
              <input {...register('delivery_date')} type="date" className="input" />
            </Field>
          </div>

          {/* Specifications */}
          <div className="card card-body grid grid-cols-1 md:grid-cols-2 gap-4">
            <h2 className="section-title col-span-full">Specifications</h2>
            <Field label="Dimensions"><input {...register('dimensions')} className="input" placeholder='L×W×H in mm' /></Field>
            <Field label="Finish"><input {...register('finish')} className="input" placeholder="Matte / Gloss" /></Field>
            <Field label="Material Details">
              <textarea {...register('material_details')} rows={2} className="input resize-none" placeholder="Oak veneer, steel frame…" />
            </Field>
            <Field label="Color Details">
              <textarea {...register('color_details')} rows={2} className="input resize-none" placeholder="RAL 9010 / Walnut" />
            </Field>
            <Field label="Upholstery"><input {...register('upholstery')} className="input" placeholder="Fabric / Leather" /></Field>
            <Field label="Specifications">
              <textarea {...register('specifications')} rows={2} className="input resize-none" />
            </Field>
            <Field label="Delivery Address">
              <textarea {...register('delivery_address')} rows={2} className="input resize-none col-span-full" />
            </Field>
            <Field label="Remarks">
              <textarea {...register('remarks')} rows={2} className="input resize-none col-span-full" />
            </Field>
          </div>

          {/* File links */}
          <div className="card card-body grid grid-cols-1 md:grid-cols-2 gap-4">
            <h2 className="section-title col-span-full">File Links (URLs or S3 paths)</h2>
            <Field label="Cover Image URL"><input {...register('cover_image_url')} className="input" placeholder="https://…" /></Field>
            <Field label="CAD Files URL"><input {...register('cad_files_url')} className="input" placeholder="https://…" /></Field>
            <Field label="Drawings URL"><input {...register('drawings_url')} className="input" placeholder="https://…" /></Field>
            <Field label="Job Cards URL"><input {...register('job_cards_url')} className="input" placeholder="https://…" /></Field>
            <Field label="Render Files URL"><input {...register('render_files_url')} className="input" placeholder="https://…" /></Field>
          </div>

          <div className="sticky bottom-0 bg-white border-t pt-4 flex justify-end gap-3">  
            <button type="button" onClick={onClose} className="btn-secondary">
                Cancel
            </button>
            <button type="submit" disabled={isPending} className="btn-primary">
              {isPending ? 'Creating…' : 'Create Project'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}