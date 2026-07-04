import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { useAuth } from '@/context/AuthContext'
import { Eye, EyeOff, Lock, Mail } from 'lucide-react'

interface FormValues { email: string; password: string }

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [showPass, setShowPass] = useState(false)
  const [serverError, setServerError] = useState('')

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormValues>()

  const onSubmit = async ({ email, password }: FormValues) => {
    try {
      setServerError('')
      await login(email, password)
      navigate('/dashboard', { replace: true })
    } catch {
      setServerError('Invalid email or password.')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo / title */}
        <div className="text-center mb-8">
          <div className="text-black font-bold text-2xl mt-1">Crafted Oak & Ore Pvt. Ltd.</div>
          {/* <h1 className="text-2xl font-bold text-white">Production Management System</h1> */}
          <p className="text-brand-300 font-bold text-sm mt-1">Production Management System</p>
        </div>

        <div className="card">
          <div className="card-body">
            {serverError && (
              <div className="mb-4 rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
                {serverError}
              </div>
            )}

            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div>
                <label className="label">Email address</label>
                <div className="relative">
                  <Mail size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    {...register('email', {
                      required: 'Email is required',
                      pattern: { value: /^\S+@\S+\.\S+$/, message: 'Enter a valid email' },
                    })}
                    type="email"
                    placeholder="you@company.com"
                    className="input pl-9"
                    autoComplete="email"
                  />
                </div>
                {errors.email && <p className="field-error">{errors.email.message}</p>}
              </div>

              <div>
                <label className="label">Password</label>
                <div className="relative">
                  <Lock size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    {...register('password', { required: 'Password is required' })}
                    type={showPass ? 'text' : 'password'}
                    placeholder="••••••••"
                    className="input pl-9 pr-10"
                    autoComplete="current-password"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPass(!showPass)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                  >
                    {showPass ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
                {errors.password && <p className="field-error">{errors.password.message}</p>}
              </div>

              <button type="submit" disabled={isSubmitting} className="btn-primary w-full mt-2 py-2.5">
                {isSubmitting ? 'Signing in…' : 'Sign in'}
              </button>
            </form>
          </div>
        </div>

        <p className="text-center text-xs text-brand-400 mt-6">
          © 2026 Crafted Oak & Ore
        </p>
      </div>
    </div>
  )
}
